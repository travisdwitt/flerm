package canvas

import "slices"

type Connection struct {
	FromID    int
	ToID      int
	FromX     int
	FromY     int
	ToX       int
	ToY       int
	Waypoints []Point
	ArrowFrom bool
	ArrowTo   bool
	Color     int
}

func cloneConn(conn Connection) Connection {
	conn.Waypoints = slices.Clone(conn.Waypoints)
	return conn
}

func (c *Canvas) FindNearestPointOnConnection(cursorX, cursorY int) (int, int, int) {
	bestDist, bestConnIdx := -1, -1
	bestX, bestY := -1, -1

	for i, conn := range c.connections {
		x, y := findNearestPointOnPath(cursorX, cursorY, connPoints(conn))
		if dist := manhattan(x, y, cursorX, cursorY); bestDist == -1 || dist < bestDist {
			bestDist, bestConnIdx = dist, i
			bestX, bestY = x, y
		}
	}

	if bestDist != -1 && bestDist <= 2 {
		return bestConnIdx, bestX, bestY
	}
	return -1, -1, -1
}

func manhattan(x1, y1, x2, y2 int) int { return abs(x1-x2) + abs(y1-y2) }

func clamp(v, lo, hi int) int { return min(max(v, min(lo, hi)), max(lo, hi)) }

func closestOnSegment(a, b Point, cursorX, cursorY int) (int, int) {
	switch {
	case a.X == b.X:
		return a.X, clamp(cursorY, a.Y, b.Y)
	case a.Y == b.Y:
		return clamp(cursorX, a.X, b.X), a.Y
	}
	corner := Point{b.X, a.Y}
	x1, y1 := closestOnSegment(a, corner, cursorX, cursorY)
	x2, y2 := closestOnSegment(corner, b, cursorX, cursorY)
	if manhattan(x1, y1, cursorX, cursorY) < manhattan(x2, y2, cursorX, cursorY) {
		return x1, y1
	}
	return x2, y2
}

func findNearestPointOnPath(x, y int, pathPoints []Point) (int, int) {
	bestX, bestY := x, y
	bestDist := -1
	for i := 0; i < len(pathPoints)-1; i++ {
		segX, segY := closestOnSegment(pathPoints[i], pathPoints[i+1], x, y)
		if dist := manhattan(segX, segY, x, y); bestDist == -1 || dist < bestDist {
			bestDist = dist
			bestX, bestY = segX, segY
		}
	}
	return bestX, bestY
}

func (c *Canvas) FindNearestEdgePoint(box Box, cursorX, cursorY int) (int, int) {
	right, bottom := box.X+box.Width-1, box.Y+box.Height-1
	clampedX := clamp(cursorX, box.X, right)
	clampedY := clamp(cursorY, box.Y, bottom)

	edgeX, edgeY, minDist := box.X, clampedY, abs(cursorX-box.X)
	for _, cand := range []struct {
		dist, x, y int
	}{
		{abs(cursorX - right), right, clampedY},
		{abs(cursorY - box.Y), clampedX, box.Y},
		{abs(cursorY - bottom), clampedX, bottom},
	} {
		if cand.dist < minDist {
			minDist, edgeX, edgeY = cand.dist, cand.x, cand.y
		}
	}
	return edgeX, edgeY
}

func center(box Box) (int, int) { return box.X + box.Width/2, box.Y + box.Height/2 }

func (c *Canvas) calculateConnectionPointsPreservingOrientation(fromID, toID int, preferHorizontal bool) (fromX, fromY, toX, toY int) {
	fromBox, okFrom := c.box(fromID)
	toBox, okTo := c.box(toID)
	if !okFrom || !okTo {
		return 0, 0, 0, 0
	}
	fcx, fcy := center(fromBox)
	tcx, tcy := center(toBox)

	if preferHorizontal {
		dir := 1
		if fcx >= tcx {
			dir = -1
		}
		return faceX(fromBox, dir), fcy, faceX(toBox, -dir), tcy
	}
	dir := 1
	if fcy >= tcy {
		dir = -1
	}
	return fcx, faceY(fromBox, dir), tcx, faceY(toBox, -dir)
}

func faceX(box Box, dir int) int {
	if dir > 0 {
		return box.X + box.Width - 1
	}
	return box.X
}

func faceY(box Box, dir int) int {
	if dir > 0 {
		return box.Y + box.Height - 1
	}
	return box.Y
}

func (c *Canvas) AddConnection(fromID, toID int) {
	fromBox, okFrom := c.box(fromID)
	toBox, okTo := c.box(toID)
	if !okFrom || !okTo {
		return
	}
	fcx, fcy := center(fromBox)
	tcx, tcy := center(toBox)
	fromX, fromY, toX, toY := c.calculateConnectionPointsPreservingOrientation(fromID, toID, abs(fcx-tcx) > abs(fcy-tcy))
	c.connections = append(c.connections, Connection{
		FromID: fromID, ToID: toID,
		FromX: fromX, FromY: fromY, ToX: toX, ToY: toY,
		Color: -1,
	})
}

func (c *Canvas) AddConnectionWithWaypoints(fromID, toID, fromX, fromY, toX, toY int, waypoints []Point) (Connection, bool) {
	if fromID >= len(c.boxes) || toID >= len(c.boxes) {
		return Connection{}, false
	}
	conn := Connection{
		FromID: fromID, ToID: toID,
		FromX: fromX, FromY: fromY, ToX: toX, ToY: toY,
		Waypoints: waypoints,
		ArrowTo:   true,
		Color:     -1,
	}
	c.connections = append(c.connections, conn)
	return conn, true
}

func (c *Canvas) RemoveSpecificConnection(target Connection) {
	c.connections = slices.DeleteFunc(c.connections, func(conn Connection) bool {
		return connectionsEqual(conn, target)
	})
}

func (c *Canvas) CycleConnectionArrowState(connIdx int) {
	if connIdx < 0 || connIdx >= len(c.connections) {
		return
	}
	conn := &c.connections[connIdx]
	switch {
	case !conn.ArrowFrom && !conn.ArrowTo:
		conn.ArrowTo = true
	case !conn.ArrowFrom:
		conn.ArrowFrom, conn.ArrowTo = true, false
	case !conn.ArrowTo:
		conn.ArrowTo = true
	default:
		conn.ArrowFrom, conn.ArrowTo = false, false
	}
}

func connectionsEqual(a, b Connection) bool {
	return a.FromID == b.FromID && a.ToID == b.ToID &&
		a.FromX == b.FromX && a.FromY == b.FromY &&
		a.ToX == b.ToX && a.ToY == b.ToY &&
		slices.Equal(a.Waypoints, b.Waypoints)
}

func (c *Canvas) RestoreConnection(connection Connection) {
	c.connections = append(c.connections, connection)
}

type connectionPathInfo struct {
	connIdx int
	points  []Point
}

func (c *Canvas) updateBranchConnections(updated []connectionPathInfo) {
	for i := range c.connections {
		conn := &c.connections[i]
		for _, from := range []bool{true, false} {
			id, other, x, y := conn.ToID, conn.FromID, &conn.ToX, &conn.ToY
			if from {
				id, other, x, y = conn.FromID, conn.ToID, &conn.FromX, &conn.FromY
			}
			if id >= 0 {
				continue
			}
			for _, u := range updated {
				if u.connIdx == i || !pointWasOnPath(*x, *y, u.points) {
					continue
				}
				newX, newY := findNearestPointOnPath(*x, *y, connPoints(c.connections[u.connIdx]))
				if newX == *x && newY == *y {
					break
				}
				if !adjustEndpointKeepingPath(conn, from, newX-*x, newY-*y) {
					if box, ok := c.box(other); ok {
						if from {
							conn.Waypoints = c.createFlexibleWaypointsForLineConnection(conn, nil, &box)
						} else {
							conn.Waypoints = c.createFlexibleWaypointsForLineConnection(conn, &box, nil)
						}
					}
				}
				simplifyConnectionPath(conn)
				break
			}
		}
	}
}

func pointWasOnPath(x, y int, pathPoints []Point) bool {
	for i := 0; i < len(pathPoints)-1; i++ {
		if pointOnSegment(x, y, pathPoints[i], pathPoints[i+1]) {
			return true
		}
	}
	return false
}

func pointOnSegment(px, py int, a, b Point) bool {
	const tolerance = 2
	minX, maxX := minmax(a.X, b.X)
	minY, maxY := minmax(a.Y, b.Y)
	if px < minX-tolerance || px > maxX+tolerance || py < minY-tolerance || py > maxY+tolerance {
		return false
	}
	switch {
	case a.Y == b.Y:
		return abs(py-a.Y) <= tolerance
	case a.X == b.X:
		return abs(px-a.X) <= tolerance
	}
	return manhattan(px, py, a.X, a.Y) <= tolerance*2 || manhattan(px, py, b.X, b.Y) <= tolerance*2
}

func (c *Canvas) rerouteConnectionsForMovedBox(id, deltaX, deltaY int) {
	box := c.boxPtr(id)
	if box == nil || (deltaX == 0 && deltaY == 0) {
		return
	}

	var updated []connectionPathInfo

	for i := range c.connections {
		conn := &c.connections[i]
		isFrom := conn.FromID == id
		if !isFrom && conn.ToID != id {
			continue
		}
		updated = append(updated, connectionPathInfo{connIdx: i, points: connPoints(*conn)})

		ax, ay, ref := &conn.ToX, &conn.ToY, Point{conn.FromX, conn.FromY}
		if isFrom {
			ax, ay, ref = &conn.FromX, &conn.FromY, Point{conn.ToX, conn.ToY}
		}
		if n := len(conn.Waypoints); n > 0 {
			ref = conn.Waypoints[n-1]
			if isFrom {
				ref = conn.Waypoints[0]
			}
		}

		regenerate := false
		if c.anchorFacesTarget(*box, *ax+deltaX, *ay+deltaY, ref.X, ref.Y) {
			regenerate = !adjustEndpointKeepingPath(conn, isFrom, deltaX, deltaY)
		} else {
			*ax, *ay = c.reanchor(*box, *ax+deltaX, *ay+deltaY, ref.X, ref.Y)
			regenerate = true
		}

		if regenerate {
			fromBox, okFrom := c.box(conn.FromID)
			toBox, okTo := c.box(conn.ToID)
			switch {
			case okFrom && okTo:
				conn.Waypoints = c.createFlexibleWaypoints(conn, fromBox, toBox)
			case isFrom:
				conn.Waypoints = c.createFlexibleWaypointsForLineConnection(conn, box, nil)
			default:
				conn.Waypoints = c.createFlexibleWaypointsForLineConnection(conn, nil, box)
			}
		}
		simplifyConnectionPath(conn)
	}

	c.updateBranchConnections(updated)
}

func (c *Canvas) reanchor(box Box, ax, ay, targetX, targetY int) (int, int) {
	switch c.GetConnectionEdge(box, ax, ay) {
	case "left", "right":
		if targetX <= box.X {
			return box.X, ay
		}
		if targetX >= box.X+box.Width-1 {
			return box.X + box.Width - 1, ay
		}
	case "top", "bottom":
		if targetY <= box.Y {
			return ax, box.Y
		}
		if targetY >= box.Y+box.Height-1 {
			return ax, box.Y + box.Height - 1
		}
	}
	return findBestAnchorPoint(box, targetX, targetY)
}

func (c *Canvas) anchorFacesTarget(box Box, anchorX, anchorY, targetX, targetY int) bool {
	switch c.GetConnectionEdge(box, anchorX, anchorY) {
	case "right":
		return targetX >= box.X+box.Width-1
	case "left":
		return targetX <= box.X
	case "top":
		return targetY <= box.Y
	case "bottom":
		return targetY >= box.Y+box.Height-1
	default:
		return false
	}
}

func adjustEndpointKeepingPath(conn *Connection, movingFrom bool, dx, dy int) bool {
	x, y, wi := &conn.ToX, &conn.ToY, len(conn.Waypoints)-1
	if movingFrom {
		x, y, wi = &conn.FromX, &conn.FromY, 0
	}
	oldX, oldY := *x, *y
	*x += dx
	*y += dy
	if len(conn.Waypoints) == 0 {
		return false
	}
	w := &conn.Waypoints[wi]
	switch {
	case w.Y == oldY:
		w.Y = *y
	case w.X == oldX:
		w.X = *x
	default:
		return false
	}
	return true
}

func simplifyConnectionPath(conn *Connection) {
	if len(conn.Waypoints) == 0 {
		return
	}
	pts := connPoints(*conn)
	kept := pts[:1]
	for i := 1; i < len(pts)-1; i++ {
		prev, cur, next := kept[len(kept)-1], pts[i], pts[i+1]
		if cur == prev || (prev.X == cur.X && cur.X == next.X) || (prev.Y == cur.Y && cur.Y == next.Y) {
			continue
		}
		kept = append(kept, cur)
	}
	if end := pts[len(pts)-1]; end != kept[len(kept)-1] {
		kept = append(kept, end)
	}

	if len(kept) <= 2 {
		conn.Waypoints = nil
	} else {
		conn.Waypoints = slices.Clone(kept[1 : len(kept)-1])
	}
}

func findBestAnchorPoint(box Box, targetX, targetY int) (int, int) {
	cx, cy := center(box)
	dx, dy := targetX-cx, targetY-cy
	if abs(dx) > abs(dy) {
		return faceX(box, dx), clamp(targetY, box.Y, box.Y+box.Height-1)
	}
	return clamp(targetX, box.X, box.X+box.Width-1), faceY(box, dy)
}

func edgeAxis(edge string) (horiz bool, dir int) {
	switch edge {
	case "right":
		return true, 1
	case "left":
		return true, -1
	case "bottom":
		return false, 1
	case "top":
		return false, -1
	}
	return true, 0
}

func pt(horiz bool, along, perp int) Point {
	if horiz {
		return Point{along, perp}
	}
	return Point{perp, along}
}

func outward(dir, a, b, off int) int {
	if dir > 0 {
		return max(a, b) + off
	}
	return min(a, b) - off
}

func parallelClearance(horiz bool) int {
	if horiz {
		return 3
	}
	return 2
}

func (c *Canvas) createFlexibleWaypoints(conn *Connection, fromBox, toBox Box) []Point {
	if conn.FromX == conn.ToX || conn.FromY == conn.ToY {
		return nil
	}

	fromEdge := c.GetConnectionEdge(fromBox, conn.FromX, conn.FromY)
	toEdge := c.GetConnectionEdge(toBox, conn.ToX, conn.ToY)
	horiz, dir := edgeAxis(fromEdge)
	if fromEdge == "unknown" {
		return []Point{{conn.ToX, conn.FromY}}
	}

	fa, fb := along(horiz, conn.FromX, conn.FromY)
	ta, tb := along(horiz, conn.ToX, conn.ToY)
	elbow := []Point{pt(horiz, ta, fb)}

	toHoriz, toDir := edgeAxis(toEdge)
	switch {
	case toEdge == "unknown":
		return elbow

	case toHoriz == horiz && toDir != dir:
		if dir*(ta-fa) > 0 {
			mid := (fa + ta) / 2
			return []Point{pt(horiz, mid, fb), pt(horiz, mid, tb)}
		}
		off := outward(dir, fa, ta, parallelClearance(horiz))
		return []Point{pt(horiz, off, fb), pt(horiz, off, tb)}

	case toHoriz == horiz:
		fFar, tFar := boxFar(fromBox, horiz, dir), boxFar(toBox, horiz, dir)
		off := outward(dir, fFar, tFar, parallelClearance(horiz))
		return []Point{pt(horiz, off, fb), pt(horiz, off, tb)}

	default:
		if dir*(ta-fa) > 0 && toDir*(tb-fb) < 0 {
			return elbow
		}
		offA := outward(dir, fa+dir, ta+3*dir, 0)
		offB := outward(toDir, fb, tb, 2)
		return []Point{pt(horiz, offA, fb), pt(horiz, offA, offB), pt(horiz, ta, offB)}
	}
}

func along(horiz bool, x, y int) (int, int) {
	if horiz {
		return x, y
	}
	return y, x
}

func boxFar(box Box, horiz bool, dir int) int {
	if horiz {
		if dir > 0 {
			return box.X + box.Width
		}
		return box.X
	}
	if dir > 0 {
		return box.Y + box.Height
	}
	return box.Y
}

func (c *Canvas) createFlexibleWaypointsForLineConnection(conn *Connection, fromBox, toBox *Box) []Point {
	if conn.FromX == conn.ToX || conn.FromY == conn.ToY {
		return nil
	}

	if fromBox != nil {
		horiz, dir := edgeAxis(c.GetConnectionEdge(*fromBox, conn.FromX, conn.FromY))
		if dir == 0 {
			return []Point{{conn.ToX, conn.FromY}}
		}
		fa, fb := along(horiz, conn.FromX, conn.FromY)
		ta, tb := along(horiz, conn.ToX, conn.ToY)
		if dir*(ta-fa) > 0 {
			return []Point{pt(horiz, ta, fb)}
		}
		off := fa + dir*parallelClearance(horiz)
		return []Point{pt(horiz, off, fb), pt(horiz, off, tb)}
	}

	if toBox != nil {
		if edge := c.GetConnectionEdge(*toBox, conn.ToX, conn.ToY); edge == "left" || edge == "right" {
			return []Point{{conn.FromX, conn.ToY}}
		}
	}
	return []Point{{conn.ToX, conn.FromY}}
}

func (c *Canvas) GetConnectionEdge(box Box, x, y int) string {
	switch {
	case x == box.X+box.Width-1:
		return "right"
	case x == box.X:
		return "left"
	case y == box.Y+box.Height-1:
		return "bottom"
	case y == box.Y:
		return "top"
	}
	return "unknown"
}

func (c *Canvas) SnapshotConnections() []Connection {
	snap := make([]Connection, len(c.connections))
	for i, conn := range c.connections {
		snap[i] = cloneConn(conn)
	}
	return snap
}

func (c *Canvas) RestoreConnectionsSnapshot(snap []Connection) {
	if len(snap) != len(c.connections) {
		return
	}
	for i, conn := range snap {
		c.connections[i] = cloneConn(conn)
	}
}

func (c *Canvas) RestoreConnections(connections []Connection) {
	for _, orig := range connections {
		for i := range c.connections {
			if c.connections[i].FromID == orig.FromID && c.connections[i].ToID == orig.ToID {
				c.connections[i] = orig
				break
			}
		}
	}
}

func (c *Canvas) GetConnectionsForBox(boxID int) []Connection {
	var result []Connection
	for _, conn := range c.connections {
		if conn.FromID == boxID || conn.ToID == boxID {
			result = append(result, cloneConn(conn))
		}
	}
	return result
}

func (c *Canvas) isPointInBoxScreen(x, y int, excludeFromID, excludeToID int, panX, panY int) bool {
	for i, box := range c.boxes {
		if i == excludeFromID || i == excludeToID {
			continue
		}
		bx, by := box.X-panX, box.Y-panY
		if x > bx && x < bx+box.Width-1 && y > by && y < by+box.Height-1 {
			return true
		}
	}
	return false
}

func segmentCells(cells []Point, a, b Point) []Point {
	if a.X != b.X && a.Y != b.Y {
		corner := Point{b.X, a.Y}
		return segmentCells(segmentCells(cells, a, corner), corner, b)
	}
	x0, x1 := minmax(a.X, b.X)
	y0, y1 := minmax(a.Y, b.Y)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			cells = append(cells, Point{x, y})
		}
	}
	return cells
}

func (c *Canvas) GetConnectionCells(connIdx int) []Point {
	if connIdx < 0 || connIdx >= len(c.connections) {
		return nil
	}
	points := connPoints(c.connections[connIdx])
	var cells []Point
	for i := 0; i < len(points)-1; i++ {
		cells = segmentCells(cells, points[i], points[i+1])
	}
	return cells
}
