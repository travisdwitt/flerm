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

func (c *Canvas) FindNearestPointOnConnection(cursorX, cursorY int) (int, int, int) {
	bestDist, bestConnIdx := -1, -1
	bestX, bestY := -1, -1

	for i, conn := range c.connections {
		points := connPoints(conn)
		for j := 0; j < len(points)-1; j++ {
			segX, segY := closestOnSegment(points[j], points[j+1], cursorX, cursorY)
			if dist := manhattan(segX, segY, cursorX, cursorY); bestDist == -1 || dist < bestDist {
				bestDist, bestConnIdx = dist, i
				bestX, bestY = segX, segY
			}
		}
	}

	if bestDist != -1 && bestDist <= 2 {
		return bestConnIdx, bestX, bestY
	}
	return -1, -1, -1
}

func manhattan(x1, y1, x2, y2 int) int { return abs(x1-x2) + abs(y1-y2) }

func clamp(v, lo, hi int) int { return min(max(v, min(lo, hi)), max(lo, hi)) }

// closestOnSegment finds the nearest cell on an axis-aligned segment. A
// diagonal segment is treated as the L-shaped path the renderer draws for it.
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

// FindNearestEdgePoint snaps a cursor to the closest point on a box's outline.
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

func (c *Canvas) CalculateConnectionPoints(fromID, toID int) (fromX, fromY, toX, toY int) {
	if fromID < 0 || fromID >= len(c.boxes) || toID < 0 || toID >= len(c.boxes) {
		return 0, 0, 0, 0
	}
	fromBox := c.boxes[fromID]
	toBox := c.boxes[toID]
	preferHorizontal := abs((fromBox.X+fromBox.Width/2)-(toBox.X+toBox.Width/2)) > abs((fromBox.Y+fromBox.Height/2)-(toBox.Y+toBox.Height/2))
	return c.calculateConnectionPointsPreservingOrientation(fromID, toID, preferHorizontal)
}

func (c *Canvas) calculateConnectionPointsPreservingOrientation(fromID, toID int, preferHorizontal bool) (fromX, fromY, toX, toY int) {
	fromBox, okFrom := c.box(fromID)
	toBox, okTo := c.box(toID)
	if !okFrom || !okTo {
		return 0, 0, 0, 0
	}

	// Each endpoint leaves from the centre of the face pointing at the other box.
	if preferHorizontal {
		dir := 1
		if fromBox.X+fromBox.Width/2 >= toBox.X+toBox.Width/2 {
			dir = -1
		}
		return faceX(fromBox, dir), fromBox.Y + fromBox.Height/2,
			faceX(toBox, -dir), toBox.Y + toBox.Height/2
	}
	dir := 1
	if fromBox.Y+fromBox.Height/2 >= toBox.Y+toBox.Height/2 {
		dir = -1
	}
	return fromBox.X + fromBox.Width/2, faceY(fromBox, dir),
		toBox.X + toBox.Width/2, faceY(toBox, -dir)
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
	if fromID >= len(c.boxes) || toID >= len(c.boxes) {
		return
	}

	fromX, fromY, toX, toY := c.CalculateConnectionPoints(fromID, toID)

	connection := Connection{
		FromID: fromID,
		ToID:   toID,
		FromX:  fromX,
		FromY:  fromY,
		ToX:    toX,
		ToY:    toY,
		Color:  -1,
	}
	c.connections = append(c.connections, connection)
}

func (c *Canvas) AddConnectionWithWaypoints(fromID, toID, fromX, fromY, toX, toY int, waypoints []Point) {
	if fromID != -1 && fromID >= len(c.boxes) {
		return
	}
	if toID != -1 && toID >= len(c.boxes) {
		return
	}

	connection := Connection{
		FromID:    fromID,
		ToID:      toID,
		FromX:     fromX,
		FromY:     fromY,
		ToX:       toX,
		ToY:       toY,
		Waypoints: waypoints,
		ArrowFrom: false,
		ArrowTo:   true,
		Color:     -1,
	}
	c.connections = append(c.connections, connection)
}

func (c *Canvas) RemoveSpecificConnection(target Connection) {
	newConnections := make([]Connection, 0)
	for _, connection := range c.connections {
		if !c.connectionsEqual(connection, target) {
			newConnections = append(newConnections, connection)
		}
	}
	c.connections = newConnections
}

func (c *Canvas) CycleConnectionArrowState(connIdx int) {
	if connIdx < 0 || connIdx >= len(c.connections) {
		return
	}
	conn := &c.connections[connIdx]
	if !conn.ArrowFrom && !conn.ArrowTo {
		conn.ArrowTo = true
	} else if !conn.ArrowFrom && conn.ArrowTo {
		conn.ArrowFrom = true
		conn.ArrowTo = false
	} else if conn.ArrowFrom && !conn.ArrowTo {
		conn.ArrowFrom = true
		conn.ArrowTo = true
	} else {
		conn.ArrowFrom = false
		conn.ArrowTo = false
	}
}

// connectionsEqual compares geometry only; arrows and colour are ignored so a
// line can be matched after those change.
func (c *Canvas) connectionsEqual(a, b Connection) bool {
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

func (c *Canvas) updateBranchConnections(updatedConnections []connectionPathInfo) {
	for i := range c.connections {
		conn := &c.connections[i]

		if conn.FromID < 0 {
			for _, updated := range updatedConnections {
				if updated.connIdx == i {
					continue
				}
				if c.PointWasOnPath(conn.FromX, conn.FromY, updated.points) {
					updatedConn := c.connections[updated.connIdx]
					newPoints := []Point{{X: updatedConn.FromX, Y: updatedConn.FromY}}
					newPoints = append(newPoints, updatedConn.Waypoints...)
					newPoints = append(newPoints, Point{X: updatedConn.ToX, Y: updatedConn.ToY})
					newX, newY := c.findNearestPointOnPath(conn.FromX, conn.FromY, newPoints)
					if newX == conn.FromX && newY == conn.FromY {
						break
					}
					if !adjustEndpointKeepingPath(conn, true, newX-conn.FromX, newY-conn.FromY) {
						if conn.ToID >= 0 && conn.ToID < len(c.boxes) {
							toBox := c.boxes[conn.ToID]
							conn.Waypoints = c.createFlexibleWaypointsForLineConnection(conn, nil, &toBox)
						}
					}
					simplifyConnectionPath(conn)
					break
				}
			}
		}

		if conn.ToID < 0 {
			for _, updated := range updatedConnections {
				if updated.connIdx == i {
					continue
				}
				if c.PointWasOnPath(conn.ToX, conn.ToY, updated.points) {
					updatedConn := c.connections[updated.connIdx]
					newPoints := []Point{{X: updatedConn.FromX, Y: updatedConn.FromY}}
					newPoints = append(newPoints, updatedConn.Waypoints...)
					newPoints = append(newPoints, Point{X: updatedConn.ToX, Y: updatedConn.ToY})
					newX, newY := c.findNearestPointOnPath(conn.ToX, conn.ToY, newPoints)
					if newX == conn.ToX && newY == conn.ToY {
						break
					}
					if !adjustEndpointKeepingPath(conn, false, newX-conn.ToX, newY-conn.ToY) {
						if conn.FromID >= 0 && conn.FromID < len(c.boxes) {
							fromBox := c.boxes[conn.FromID]
							conn.Waypoints = c.createFlexibleWaypointsForLineConnection(conn, &fromBox, nil)
						}
					}
					simplifyConnectionPath(conn)
					break
				}
			}
		}
	}
}

func (c *Canvas) PointWasOnPath(x, y int, pathPoints []Point) bool {
	for i := 0; i < len(pathPoints)-1; i++ {
		if c.pointOnSegment(x, y, pathPoints[i].X, pathPoints[i].Y, pathPoints[i+1].X, pathPoints[i+1].Y) {
			return true
		}
	}
	return false
}

func (c *Canvas) pointOnSegment(px, py, x1, y1, x2, y2 int) bool {

	minX, maxX := min(x1, x2), max(x1, x2)
	minY, maxY := min(y1, y2), max(y1, y2)

	tolerance := 2
	if px < minX-tolerance || px > maxX+tolerance || py < minY-tolerance || py > maxY+tolerance {
		return false
	}

	if y1 == y2 {
		return abs(py-y1) <= tolerance && px >= minX-tolerance && px <= maxX+tolerance
	}

	if x1 == x2 {
		return abs(px-x1) <= tolerance && py >= minY-tolerance && py <= maxY+tolerance
	}

	dist1 := abs(px-x1) + abs(py-y1)
	dist2 := abs(px-x2) + abs(py-y2)
	return dist1 <= tolerance*2 || dist2 <= tolerance*2
}

func (c *Canvas) findNearestPointOnPath(x, y int, pathPoints []Point) (int, int) {
	bestX, bestY := x, y
	bestDist := -1

	for i := 0; i < len(pathPoints)-1; i++ {
		segX, segY := closestOnSegment(pathPoints[i], pathPoints[i+1], x, y)
		dist := abs(segX-x) + abs(segY-y)
		if bestDist == -1 || dist < bestDist {
			bestDist = dist
			bestX, bestY = segX, segY
		}
	}

	return bestX, bestY
}

func (c *Canvas) rerouteConnectionsForMovedBox(id, deltaX, deltaY int) {
	if id < 0 || id >= len(c.boxes) || (deltaX == 0 && deltaY == 0) {
		return
	}
	box := &c.boxes[id]

	var updatedConnections []connectionPathInfo

	for i := range c.connections {
		conn := &c.connections[i]

		isFromThisBox := conn.FromID == id
		isToThisBox := conn.ToID == id
		if !isFromThisBox && !isToThisBox {
			continue
		}

		oldPoints := []Point{{X: conn.FromX, Y: conn.FromY}}
		oldPoints = append(oldPoints, conn.Waypoints...)
		oldPoints = append(oldPoints, Point{X: conn.ToX, Y: conn.ToY})
		updatedConnections = append(updatedConnections, connectionPathInfo{connIdx: i, points: oldPoints})

		fromIsValidBox := conn.FromID >= 0 && conn.FromID < len(c.boxes)
		toIsValidBox := conn.ToID >= 0 && conn.ToID < len(c.boxes)

		regenerate := false
		if isFromThisBox {
			refX, refY := conn.ToX, conn.ToY
			if len(conn.Waypoints) > 0 {
				refX, refY = conn.Waypoints[0].X, conn.Waypoints[0].Y
			}
			if c.anchorFacesTarget(*box, conn.FromX+deltaX, conn.FromY+deltaY, refX, refY) {
				if !adjustEndpointKeepingPath(conn, true, deltaX, deltaY) {
					regenerate = true
				}
			} else {
				conn.FromX, conn.FromY = c.reanchor(*box, conn.FromX+deltaX, conn.FromY+deltaY, refX, refY)
				regenerate = true
			}
		} else {
			refX, refY := conn.FromX, conn.FromY
			if len(conn.Waypoints) > 0 {
				refX, refY = conn.Waypoints[len(conn.Waypoints)-1].X, conn.Waypoints[len(conn.Waypoints)-1].Y
			}
			if c.anchorFacesTarget(*box, conn.ToX+deltaX, conn.ToY+deltaY, refX, refY) {
				if !adjustEndpointKeepingPath(conn, false, deltaX, deltaY) {
					regenerate = true
				}
			} else {
				conn.ToX, conn.ToY = c.reanchor(*box, conn.ToX+deltaX, conn.ToY+deltaY, refX, refY)
				regenerate = true
			}
		}

		if regenerate {
			if fromIsValidBox && toIsValidBox {
				conn.Waypoints = c.createFlexibleWaypoints(conn, c.boxes[conn.FromID], c.boxes[conn.ToID])
			} else if isFromThisBox {
				conn.Waypoints = c.createFlexibleWaypointsForLineConnection(conn, box, nil)
			} else {
				conn.Waypoints = c.createFlexibleWaypointsForLineConnection(conn, nil, box)
			}
		}
		simplifyConnectionPath(conn)
	}

	c.updateBranchConnections(updatedConnections)
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
	return c.findBestAnchorPoint(box, targetX, targetY)
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
	if movingFrom {
		oldX, oldY := conn.FromX, conn.FromY
		conn.FromX += dx
		conn.FromY += dy
		if len(conn.Waypoints) == 0 {
			return false
		}
		w := &conn.Waypoints[0]
		switch {
		case w.Y == oldY:
			w.Y = conn.FromY
		case w.X == oldX:
			w.X = conn.FromX
		default:
			return false
		}
		return true
	}

	oldX, oldY := conn.ToX, conn.ToY
	conn.ToX += dx
	conn.ToY += dy
	if len(conn.Waypoints) == 0 {
		return false
	}
	w := &conn.Waypoints[len(conn.Waypoints)-1]
	switch {
	case w.Y == oldY:
		w.Y = conn.ToY
	case w.X == oldX:
		w.X = conn.ToX
	default:
		return false
	}
	return true
}

func simplifyConnectionPath(conn *Connection) {
	if len(conn.Waypoints) == 0 {
		return
	}
	pts := make([]Point, 0, len(conn.Waypoints)+2)
	pts = append(pts, Point{X: conn.FromX, Y: conn.FromY})
	pts = append(pts, conn.Waypoints...)
	pts = append(pts, Point{X: conn.ToX, Y: conn.ToY})

	kept := pts[:1]
	for i := 1; i < len(pts)-1; i++ {
		prev := kept[len(kept)-1]
		cur := pts[i]
		next := pts[i+1]
		if cur == prev {
			continue
		}
		if (prev.X == cur.X && cur.X == next.X) || (prev.Y == cur.Y && cur.Y == next.Y) {
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
		conn.Waypoints = append([]Point(nil), kept[1:len(kept)-1]...)
	}
}

func (c *Canvas) findBestAnchorPoint(box Box, targetX, targetY int) (int, int) {
	boxCenterX := box.X + box.Width/2
	boxCenterY := box.Y + box.Height/2

	dx := targetX - boxCenterX
	dy := targetY - boxCenterY

	if abs(dx) > abs(dy) {

		if dx > 0 {

			y := targetY
			if y < box.Y {
				y = box.Y
			} else if y >= box.Y+box.Height {
				y = box.Y + box.Height - 1
			}
			return box.X + box.Width - 1, y
		} else {

			y := targetY
			if y < box.Y {
				y = box.Y
			} else if y >= box.Y+box.Height {
				y = box.Y + box.Height - 1
			}
			return box.X, y
		}
	} else {

		if dy > 0 {

			x := targetX
			if x < box.X {
				x = box.X
			} else if x >= box.X+box.Width {
				x = box.X + box.Width - 1
			}
			return x, box.Y + box.Height - 1
		} else {

			x := targetX
			if x < box.X {
				x = box.X
			} else if x >= box.X+box.Width {
				x = box.X + box.Width - 1
			}
			return x, box.Y
		}
	}
}

// Edges are named "left"/"right"/"top"/"bottom". Routing is written once in
// along/perpendicular coordinates: "along" is the axis the source edge faces,
// so a vertical source edge is just the horizontal case with X and Y swapped.

// edgeAxis reports whether an edge faces along X, and which way it points.
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

// pt rebuilds a Point from along/perpendicular coordinates.
func pt(horiz bool, along, perp int) Point {
	if horiz {
		return Point{along, perp}
	}
	return Point{perp, along}
}

// outward returns whichever of a, b lies further in direction dir, offset by
// dir*off — i.e. a point clear of both on that side.
func outward(dir, a, b, off int) int {
	if dir > 0 {
		return max(a, b) + off
	}
	return min(a, b) - off
}

// parallelClearance is how far a line detours sideways when both endpoints face
// the same axis; rows are cheaper to spend than columns.
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

	case toHoriz == horiz && toDir != dir: // facing each other
		if dir*(ta-fa) > 0 {
			mid := (fa + ta) / 2
			return []Point{pt(horiz, mid, fb), pt(horiz, mid, tb)}
		}
		off := outward(dir, fa, ta, parallelClearance(horiz))
		return []Point{pt(horiz, off, fb), pt(horiz, off, tb)}

	case toHoriz == horiz: // both facing the same way: go around the far box
		fFar, tFar := boxFar(fromBox, horiz, dir), boxFar(toBox, horiz, dir)
		off := outward(dir, fFar, tFar, parallelClearance(horiz))
		return []Point{pt(horiz, off, fb), pt(horiz, off, tb)}

	default: // perpendicular edges
		if dir*(ta-fa) > 0 && toDir*(tb-fb) < 0 {
			return elbow
		}
		offA := outward(dir, fa+dir, ta+3*dir, 0)
		offB := outward(toDir, fb, tb, 2)
		return []Point{pt(horiz, offA, fb), pt(horiz, offA, offB), pt(horiz, ta, offB)}
	}
}

// along splits a point into (source-edge axis, perpendicular axis) coordinates.
func along(horiz bool, x, y int) (int, int) {
	if horiz {
		return x, y
	}
	return y, x
}

// boxFar returns the box's bounding coordinate on the side dir points to.
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
	if x == box.X+box.Width-1 {
		return "right"
	} else if x == box.X {
		return "left"
	} else if y == box.Y+box.Height-1 {
		return "bottom"
	} else if y == box.Y {
		return "top"
	}
	return "unknown"
}

func (c *Canvas) SnapshotConnections() []Connection {
	snap := make([]Connection, len(c.connections))
	for i, conn := range c.connections {
		snap[i] = conn
		if len(conn.Waypoints) > 0 {
			snap[i].Waypoints = append([]Point(nil), conn.Waypoints...)
		} else {
			snap[i].Waypoints = nil
		}
	}
	return snap
}

func (c *Canvas) RestoreConnectionsSnapshot(snap []Connection) {
	if len(snap) != len(c.connections) {
		return
	}
	for i, conn := range snap {
		c.connections[i] = conn
		if len(conn.Waypoints) > 0 {
			c.connections[i].Waypoints = append([]Point(nil), conn.Waypoints...)
		} else {
			c.connections[i].Waypoints = nil
		}
	}
}

func (c *Canvas) RestoreConnections(connections []Connection) {
	for _, origConn := range connections {

		for i := range c.connections {
			if c.connections[i].FromID == origConn.FromID && c.connections[i].ToID == origConn.ToID {

				c.connections[i] = origConn
				break
			}
		}
	}
}

func (c *Canvas) GetConnectionsForBox(boxID int) []Connection {
	var result []Connection
	for _, conn := range c.connections {
		if conn.FromID == boxID || conn.ToID == boxID {
			connCopy := Connection{
				FromID:    conn.FromID,
				ToID:      conn.ToID,
				FromX:     conn.FromX,
				FromY:     conn.FromY,
				ToX:       conn.ToX,
				ToY:       conn.ToY,
				ArrowFrom: conn.ArrowFrom,
				ArrowTo:   conn.ArrowTo,
				Waypoints: make([]Point, len(conn.Waypoints)),
				Color:     conn.Color,
			}
			copy(connCopy.Waypoints, conn.Waypoints)
			result = append(result, connCopy)
		}
	}
	return result
}

func (c *Canvas) isPointInBoxScreen(x, y int, excludeFromID, excludeToID int, panX, panY int) bool {
	for i, box := range c.boxes {
		boxScreenX := box.X - panX
		boxScreenY := box.Y - panY
		if x > boxScreenX && x < boxScreenX+box.Width-1 && y > boxScreenY && y < boxScreenY+box.Height-1 {
			if i != excludeFromID && i != excludeToID {
				return true
			}
		}
	}
	return false
}

func (c *Canvas) GetConnectionCells(connIdx int) []Point {
	if connIdx < 0 || connIdx >= len(c.connections) {
		return nil
	}
	conn := c.connections[connIdx]
	cells := make([]Point, 0)
	points := []Point{{conn.FromX, conn.FromY}}
	points = append(points, conn.Waypoints...)
	points = append(points, Point{conn.ToX, conn.ToY})
	for i := 0; i < len(points)-1; i++ {
		from := points[i]
		to := points[i+1]
		if from.X == to.X {
			startY, endY := from.Y, to.Y
			if startY > endY {
				startY, endY = endY, startY
			}
			for y := startY; y <= endY; y++ {
				cells = append(cells, Point{X: from.X, Y: y})
			}
		} else if from.Y == to.Y {
			startX, endX := from.X, to.X
			if startX > endX {
				startX, endX = endX, startX
			}
			for x := startX; x <= endX; x++ {
				cells = append(cells, Point{X: x, Y: from.Y})
			}
		} else {
			cornerX := to.X
			cornerY := from.Y
			startX, endX := from.X, cornerX
			if startX > endX {
				startX, endX = endX, startX
			}
			for x := startX; x <= endX; x++ {
				cells = append(cells, Point{X: x, Y: from.Y})
			}
			startY, endY := cornerY, to.Y
			if startY > endY {
				startY, endY = endY, startY
			}
			for y := startY; y <= endY; y++ {
				cells = append(cells, Point{X: cornerX, Y: y})
			}
		}
	}
	return cells
}
