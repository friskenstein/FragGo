package game

import (
	"fmt"
	"time"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/window"
	"github.com/go-gl/glfw/v3.3/glfw"
)

const (
	windowWidth  = 1600
	windowHeight = 900

	mouseSensitivity = 0.0024
	gravity          = 34.0
	jumpVelocity     = 12.5
	moveSpeed        = 16.0
	boostSpeed       = 22.0
	airControl       = 5.0
	weaponDamage     = 34
	weaponRange      = 90.0
	playerRadius     = 0.6
	playerHeight     = 1.8
	playerEyeHeight  = 1.55
	worldLimit       = 46.0
	stepUpHeight     = 0.7
	stepDownHeight   = 0.85
	groundSnapHeight = 0.35
	spawnX           = 0.0
	spawnY           = 0.0
	spawnZ           = 34.0
	collisionEpsilon = 0.01
)

type Game struct {
	win      *window.GlfwWindow
	renderer *renderer.Renderer
	keys     *window.KeyState
	scene    *core.Node
	camera   *camera.Camera

	playerRoot    *core.Node
	phase         gamePhase
	sessionMode   sessionMode
	matchConfig   matchConfig
	menuSelection int

	playerPos      math32.Vector3
	playerVelocity math32.Vector3
	playerGrounded bool
	jumpHeld       bool
	yaw            float32
	pitch          float32

	mouseCaptured bool
	cursorSeeded  bool
	cursorX       float32
	cursorY       float32

	fireQueued      bool
	fireCooldown    time.Duration
	matchTime       time.Duration
	roundElapsed    time.Duration
	frags           int
	playerDeaths    int
	shotsFired      int
	shotsHit        int
	statusText      string
	statusTTL       time.Duration
	menuCameraAngle float32

	platforms  []platform
	colliders  []boxCollider
	combatants []*combatant

	infoLabel      *gui.Label
	controlsLabel  *gui.Label
	crosshair      *gui.Label
	statusLabel    *gui.Label
	logoImage      *gui.Image
	logoAspect     float32
	menuTitleLabel *gui.Label
	menuBodyLabel  *gui.Label
	rosterLabel    *gui.Label

	cameraTrace *graphic.Lines
	muzzleTrace *graphic.Lines
	impactFlash *graphic.Mesh
	traceTTL    time.Duration
	impactTTL   time.Duration
}

func New() (*Game, error) {

	if err := window.Init(windowWidth, windowHeight, "Go3D Arena Prototype"); err != nil {
		return nil, err
	}

	win := window.Get().(*window.GlfwWindow)
	win.SetSwapInterval(1)

	rend := renderer.NewRenderer(win.Gls())
	if err := rend.AddDefaultShaders(); err != nil {
		win.Destroy()
		return nil, err
	}

	g := &Game{
		win:      win,
		renderer: rend,
		keys:     window.NewKeyState(win),
		scene:    core.NewNode(),
		camera:   camera.NewPerspective(float32(windowWidth)/float32(windowHeight), 0.1, 300, 78, camera.Vertical),
	}

	g.playerPos.Set(spawnX, spawnY, spawnZ)
	g.playerGrounded = true
	g.yaw = 0
	g.configureMenuDefaults()

	g.scene.Add(g.camera)
	gui.Manager().Set(g.scene)

	g.subscribeEvents()
	if err := g.buildWorld(); err != nil {
		g.shutdown()
		return nil, err
	}
	g.buildHUD()
	g.buildEffects()
	g.releaseMouse()
	g.setStatus("Configure a hosted match and press Enter", 5*time.Second)
	g.onResize("", nil)
	g.refreshHUD()

	return g, nil
}

func (g *Game) Run() {

	defer g.shutdown()

	lastFrame := time.Now()
	for !g.win.ShouldClose() {
		now := time.Now()
		delta := now.Sub(lastFrame)
		lastFrame = now

		if delta > 50*time.Millisecond {
			delta = 50 * time.Millisecond
		}

		g.update(delta)

		g.win.SwapBuffers()
		g.win.PollEvents()
	}
}

func (g *Game) shutdown() {

	if g.keys != nil {
		g.keys.Dispose()
	}
	if g.win != nil {
		g.win.Destroy()
	}
}

func (g *Game) subscribeEvents() {

	g.win.Subscribe(window.OnWindowSize, g.onResize)
	g.win.Subscribe(window.OnCursor, g.onCursor)
	g.win.Subscribe(window.OnMouseDown, g.onMouseDown)
	g.win.Subscribe(window.OnKeyDown, g.onKeyDown)
	g.win.Subscribe(window.OnWindowFocus, g.onFocus)
}

func (g *Game) update(delta time.Duration) {

	if g.statusTTL > 0 {
		g.statusTTL -= delta
		if g.statusTTL <= 0 {
			g.statusTTL = 0
			g.statusText = ""
		}
	}

	if g.phase == phaseMatch {
		g.roundElapsed += delta
		speedMultiplier := g.currentSpeedMultiplier()
		gameDelta := scaleDuration(delta, speedMultiplier)
		g.matchTime += gameDelta

		if g.fireCooldown > 0 {
			g.fireCooldown -= gameDelta
			if g.fireCooldown < 0 {
				g.fireCooldown = 0
			}
		}

		g.updatePlayer(float32(gameDelta.Seconds()))
		g.updateCombatants(gameDelta)
		if g.roundElapsed >= g.matchConfig.RoundDuration {
			g.endMatch()
		}
	} else {
		g.menuCameraAngle += float32(delta.Seconds()) * 0.18
	}

	g.updateCamera()
	g.updateEffects(delta)

	if g.phase == phaseMatch && g.fireQueued {
		g.fireQueued = false
		g.fireWeapon()
	}

	g.refreshHUD()
	g.render()
}

func (g *Game) updatePlayer(dt float32) {

	boosting := g.keys.Pressed(window.KeyLeftShift) || g.keys.Pressed(window.KeyRightShift)
	currentMoveSpeed := float32(moveSpeed)
	if boosting {
		currentMoveSpeed = boostSpeed
	}

	moveInput := math32.Vector3{}
	if g.keys.Pressed(window.KeyW) {
		moveInput.Z += 1
	}
	if g.keys.Pressed(window.KeyS) {
		moveInput.Z -= 1
	}
	if g.keys.Pressed(window.KeyA) {
		moveInput.X -= 1
	}
	if g.keys.Pressed(window.KeyD) {
		moveInput.X += 1
	}
	if moveInput.LengthSq() > 0 {
		moveInput.Normalize()
	}

	forwardFlat := math32.Vector3{X: math32.Sin(g.yaw), Y: 0, Z: -math32.Cos(g.yaw)}
	rightFlat := math32.Vector3{X: math32.Cos(g.yaw), Y: 0, Z: math32.Sin(g.yaw)}

	var wishDir math32.Vector3
	wishDir.Add(forwardFlat.MultiplyScalar(moveInput.Z))
	wishDir.Add(rightFlat.MultiplyScalar(moveInput.X))
	if wishDir.LengthSq() > 0 {
		wishDir.Normalize()
	}

	if g.playerGrounded {
		g.playerVelocity.X = wishDir.X * currentMoveSpeed
		g.playerVelocity.Z = wishDir.Z * currentMoveSpeed
	} else {
		targetX := wishDir.X * currentMoveSpeed
		targetZ := wishDir.Z * currentMoveSpeed
		blend := math32.Clamp(dt*airControl, 0, 1)
		g.playerVelocity.X += (targetX - g.playerVelocity.X) * blend
		g.playerVelocity.Z += (targetZ - g.playerVelocity.Z) * blend
		g.playerVelocity.Y -= gravity * dt
	}

	jumpPressed := g.keys.Pressed(window.KeySpace)
	if jumpPressed && !g.jumpHeld && g.playerGrounded {
		g.playerGrounded = false
		g.playerVelocity.Y = jumpVelocity
	}
	g.jumpHeld = jumpPressed

	previousY := g.playerPos.Y
	g.playerPos.Y += g.playerVelocity.Y * dt
	g.movePlayerHorizontal(g.playerVelocity.X*dt, g.playerVelocity.Z*dt)

	g.playerPos.X = math32.Clamp(g.playerPos.X, -worldLimit, worldLimit)
	g.playerPos.Z = math32.Clamp(g.playerPos.Z, -worldLimit, worldLimit)

	landed, supportY := g.resolveGround(previousY)
	g.playerGrounded = landed
	if landed {
		g.playerPos.Y = supportY
		if g.playerVelocity.Y < 0 {
			g.playerVelocity.Y = 0
		}
	}

	if g.playerPos.Y < -10 {
		g.playerDeaths++
		g.resetPlayer()
		g.setStatus("Respawned at south spawn", 2*time.Second)
	}

	g.syncPlayerModel()
}

func (g *Game) resolveGround(previousY float32) (bool, float32) {

	bestSupport := float32(0)
	landed := false
	for _, platform := range g.platforms {
		if !platform.contains(g.playerPos.X, g.playerPos.Z, playerRadius) {
			continue
		}

		top := platform.top()
		canLand := g.playerVelocity.Y <= 0 &&
			previousY >= top-0.1 &&
			g.playerPos.Y <= top+groundSnapHeight
		canStep := g.playerGrounded &&
			top >= previousY-stepDownHeight &&
			top <= previousY+stepUpHeight
		if !canLand && !canStep {
			continue
		}

		if !landed || top > bestSupport {
			bestSupport = top
			landed = true
		}
	}

	return landed, bestSupport
}

func (g *Game) movePlayerHorizontal(deltaX, deltaZ float32) {

	g.playerPos.X, g.playerVelocity.X = g.resolvePlayerAxis(g.playerPos, deltaX, true, g.playerVelocity.X)

	intermediate := g.playerPos
	g.playerPos.Z, g.playerVelocity.Z = g.resolvePlayerAxis(intermediate, deltaZ, false, g.playerVelocity.Z)
}

func (g *Game) resolvePlayerAxis(base math32.Vector3, delta float32, moveX bool, velocity float32) (float32, float32) {

	if math32.Abs(delta) < 0.00001 {
		if moveX {
			return base.X, velocity
		}
		return base.Z, velocity
	}

	target := base
	if moveX {
		target.X += delta
	} else {
		target.Z += delta
	}

	for idx := range g.colliders {
		collider := g.colliders[idx]
		if !g.colliderBlocksPlayerAt(collider, target) {
			continue
		}

		min := collider.min()
		max := collider.max()
		if moveX {
			if delta > 0 {
				target.X = min.X - playerRadius - collisionEpsilon
			} else {
				target.X = max.X + playerRadius + collisionEpsilon
			}
		} else {
			if delta > 0 {
				target.Z = min.Z - playerRadius - collisionEpsilon
			} else {
				target.Z = max.Z + playerRadius + collisionEpsilon
			}
		}
		velocity = 0
	}

	if moveX {
		return target.X, velocity
	}
	return target.Z, velocity
}

func (g *Game) colliderBlocksPlayerAt(collider boxCollider, pos math32.Vector3) bool {

	playerBottom := pos.Y
	playerTop := pos.Y + playerHeight
	if collider.top() <= playerBottom+0.05 || collider.bottom() >= playerTop-0.05 {
		return false
	}

	if collider.walkable && g.playerGrounded {
		top := collider.top()
		if top >= playerBottom-groundSnapHeight && top <= playerBottom+stepUpHeight {
			return false
		}
	}

	min := collider.min()
	max := collider.max()
	return pos.X >= min.X-playerRadius &&
		pos.X <= max.X+playerRadius &&
		pos.Z >= min.Z-playerRadius &&
		pos.Z <= max.Z+playerRadius
}

func (g *Game) resetPlayer() {

	g.playerPos.Set(spawnX, spawnY, spawnZ)
	g.playerVelocity.Set(0, 0, 0)
	g.playerGrounded = true
	g.pitch = 0
	g.yaw = 0
}

func (g *Game) updateCamera() {

	if g.phase != phaseMatch {
		radius := float32(66)
		camPos := math32.Vector3{
			X: math32.Cos(g.menuCameraAngle) * radius,
			Y: 20 + math32.Sin(g.menuCameraAngle*0.55)*4,
			Z: math32.Sin(g.menuCameraAngle) * radius,
		}
		g.camera.SetFov(68)
		g.camera.SetPosition(camPos.X, camPos.Y, camPos.Z)
		g.camera.LookAt(&math32.Vector3{X: 0, Y: 4, Z: 0}, &math32.Vector3{Y: 1})
		g.playerRoot.SetVisible(false)
		return
	}

	headPos := g.playerHeadPosition()
	viewDir := g.viewDirection()
	g.camera.SetFov(74)

	backOffset := viewDir.Clone().MultiplyScalar(-6.5)
	backOffset.Y += 2.25

	camPos := headPos
	camPos.Add(backOffset)

	aimPoint := camPos
	aimPoint.Add(viewDir.Clone().MultiplyScalar(weaponRange))

	g.camera.SetPosition(camPos.X, camPos.Y, camPos.Z)
	g.camera.LookAt(&aimPoint, &math32.Vector3{Y: 1})
	g.playerRoot.SetVisible(true)
}

func (g *Game) render() {

	g.win.Gls().Enable(gls.DEPTH_TEST)
	g.win.Gls().ClearColor(0.05, 0.06, 0.09, 1.0)
	g.win.Gls().Clear(gls.COLOR_BUFFER_BIT | gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT)
	if err := g.renderer.Render(g.scene, g.camera); err != nil {
		panic(err)
	}
}

func (g *Game) fireWeapon() {

	if g.fireCooldown > 0 {
		return
	}

	g.fireCooldown = 120 * time.Millisecond
	g.shotsFired++

	cameraOrigin := g.camera.Position()
	cameraDir := g.viewDirection()
	muzzleOrigin := g.playerMuzzlePosition()
	cameraTraceStart, cameraTraceDistance := clipCameraTraceToMuzzlePlane(cameraOrigin, cameraDir, muzzleOrigin, weaponRange)
	aimPoint := cameraTraceStart
	aimPoint.Add(cameraDir.Clone().MultiplyScalar(cameraTraceDistance))

	cameraHit := g.traceScene(cameraTraceStart, cameraDir, cameraTraceDistance)
	cameraTraceEnd := aimPoint
	if cameraHit.hit {
		aimPoint = cameraHit.point
		cameraTraceEnd = cameraHit.point
	}
	g.showTrace(g.cameraTrace, cameraTraceStart, cameraTraceEnd)

	muzzleDir := aimPoint.Clone().Sub(&muzzleOrigin)
	muzzleDistance := muzzleDir.Length()
	if muzzleDistance <= 0.001 {
		muzzleDir = cameraDir.Clone()
		muzzleDistance = weaponRange
	} else {
		muzzleDir.Normalize()
	}

	shotHit := g.traceScene(muzzleOrigin, muzzleDir, muzzleDistance)
	if !shotHit.hit {
		missEnd := muzzleOrigin
		missEnd.Add(muzzleDir.Clone().MultiplyScalar(muzzleDistance))
		g.showTrace(g.muzzleTrace, muzzleOrigin, missEnd)
		g.hideImpact()
		g.setStatus("Shot wide", 400*time.Millisecond)
		return
	}
	g.showTrace(g.muzzleTrace, muzzleOrigin, shotHit.point)
	g.showImpact(shotHit.point)

	if shotHit.target == nil {
		g.setStatus(fmt.Sprintf("Shot blocked by %s", shotHit.blockerName()), 700*time.Millisecond)
		return
	}

	g.shotsHit++
	if shotHit.target.applyDamage(weaponDamage) {
		g.frags++
		g.setStatus(fmt.Sprintf("Fragged %s", shotHit.target.name), 1400*time.Millisecond)
		return
	}

	g.setStatus(fmt.Sprintf("Tagged %s (%d hp)", shotHit.target.name, shotHit.target.health), 700*time.Millisecond)
}

func (g *Game) playerHeadPosition() math32.Vector3 {

	return math32.Vector3{
		X: g.playerPos.X,
		Y: g.playerPos.Y + playerEyeHeight,
		Z: g.playerPos.Z,
	}
}

func (g *Game) playerMuzzlePosition() math32.Vector3 {

	headPos := g.playerHeadPosition()
	right := math32.Vector3{X: math32.Cos(g.yaw), Y: 0, Z: math32.Sin(g.yaw)}
	forward := math32.Vector3{X: math32.Sin(g.yaw), Y: 0, Z: -math32.Cos(g.yaw)}

	headPos.Add(right.MultiplyScalar(0.45))
	headPos.Add(forward.MultiplyScalar(0.7))
	headPos.Y -= 0.18
	return headPos
}

func (g *Game) viewDirection() *math32.Vector3 {

	cosPitch := math32.Cos(g.pitch)
	dir := &math32.Vector3{
		X: math32.Sin(g.yaw) * cosPitch,
		Y: math32.Sin(g.pitch),
		Z: -math32.Cos(g.yaw) * cosPitch,
	}
	dir.Normalize()
	return dir
}

func (g *Game) setStatus(text string, ttl time.Duration) {

	g.statusText = text
	g.statusTTL = ttl
}

func (g *Game) onResize(string, interface{}) {

	width, height := g.win.GetSize()
	fbw, fbh := g.win.GetFramebufferSize()
	g.win.Gls().Viewport(0, 0, int32(fbw), int32(fbh))
	g.camera.SetAspect(float32(width) / float32(height))
	g.layoutHUD(float32(width), float32(height))
}

func (g *Game) onCursor(_ string, ev interface{}) {

	cursor := ev.(*window.CursorEvent)
	if g.phase != phaseMatch || !g.mouseCaptured {
		g.cursorX = cursor.Xpos
		g.cursorY = cursor.Ypos
		g.cursorSeeded = true
		return
	}
	if !g.cursorSeeded {
		g.cursorX = cursor.Xpos
		g.cursorY = cursor.Ypos
		g.cursorSeeded = true
		return
	}

	dx := cursor.Xpos - g.cursorX
	dy := cursor.Ypos - g.cursorY
	g.cursorX = cursor.Xpos
	g.cursorY = cursor.Ypos

	g.yaw += dx * mouseSensitivity
	g.pitch -= dy * mouseSensitivity
	g.pitch = math32.Clamp(g.pitch, -1.2, 1.2)
}

func (g *Game) onMouseDown(_ string, ev interface{}) {

	if g.phase != phaseMatch {
		return
	}

	mouse := ev.(*window.MouseEvent)
	switch mouse.Button {
	case window.MouseButtonLeft:
		if !g.mouseCaptured {
			g.captureMouse()
			g.setStatus("Mouse capture enabled", 900*time.Millisecond)
			return
		}
		g.fireQueued = true
	}
}

func (g *Game) onKeyDown(_ string, ev interface{}) {

	key := ev.(*window.KeyEvent)
	if g.phase == phaseResults {
		g.handleResultsInput(key.Key)
		return
	}
	if g.phase != phaseMatch {
		g.handleMenuInput(key.Key)
		return
	}

	switch key.Key {
	case window.KeyEscape:
		if g.mouseCaptured {
			g.releaseMouse()
			g.setStatus("Mouse released", 900*time.Millisecond)
		} else {
			g.captureMouse()
			g.setStatus("Mouse capture enabled", 900*time.Millisecond)
		}
	case window.KeyR:
		g.resetPlayer()
		g.setStatus("Player reset", time.Second)
	case window.KeyF2:
		g.returnToMenu("Match abandoned")
	}
}

func (g *Game) onFocus(_ string, ev interface{}) {

	focus := ev.(*window.FocusEvent)
	if !focus.Focused && g.mouseCaptured {
		g.releaseMouse()
	}
}

func (g *Game) captureMouse() {

	g.mouseCaptured = true
	g.cursorSeeded = false
	g.win.SetInputMode(glfw.InputMode(window.CursorInputMode), int(window.CursorDisabled))
}

func (g *Game) releaseMouse() {

	g.mouseCaptured = false
	g.cursorSeeded = false
	g.win.SetInputMode(glfw.InputMode(window.CursorInputMode), int(window.CursorNormal))
}

func raySphereHit(origin math32.Vector3, direction *math32.Vector3, center math32.Vector3, radius float32) (float32, bool) {

	offset := origin
	offset.Sub(&center)

	a := direction.Dot(direction)
	b := float32(2) * offset.Dot(direction)
	c := offset.Dot(&offset) - radius*radius
	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return 0, false
	}

	sqrtDiscriminant := math32.Sqrt(discriminant)
	t0 := (-b - sqrtDiscriminant) / (2 * a)
	if t0 >= 0 {
		return t0, true
	}

	t1 := (-b + sqrtDiscriminant) / (2 * a)
	if t1 >= 0 {
		return t1, true
	}

	return 0, false
}
