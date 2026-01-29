package main

import "sort"

// startMindMapSiblingInput prepares to create a sibling node
func (m *model) startMindMapSiblingInput(siblingID int) {
	if siblingID < 0 || siblingID >= len(m.getCanvas().boxes) {
		return
	}
	box := m.getCanvas().boxes[siblingID]

	// Calculate position below the sibling
	m.mindMapInputX = box.X
	m.mindMapInputY = box.Y + box.Height + 1
	m.mindMapInputText = ""
	m.mindMapInputCursorPos = 0
	m.mindMapInputParent = -2 // -2 means sibling
	m.mindMapInputSibling = siblingID
	m.mode = ModeMindMapInput
}

// startMindMapChildInput prepares to create a child node
func (m *model) startMindMapChildInput(parentID int) {
	if parentID < 0 || parentID >= len(m.getCanvas().boxes) {
		return
	}
	box := m.getCanvas().boxes[parentID]

	// Calculate position to the right of parent
	// Find existing children to position below them
	children := m.getMindMapChildren(parentID)
	var childY int
	if len(children) > 0 {
		// Position below last child
		lastChild := children[len(children)-1]
		if lastChild < len(m.getCanvas().boxes) {
			lastBox := m.getCanvas().boxes[lastChild]
			childY = lastBox.Y + lastBox.Height + 1
		} else {
			childY = box.Y
		}
	} else {
		childY = box.Y
	}

	m.mindMapInputX = box.X + box.Width + 4
	m.mindMapInputY = childY
	m.mindMapInputText = ""
	m.mindMapInputCursorPos = 0
	m.mindMapInputParent = parentID
	m.mindMapInputSibling = -1
	m.mode = ModeMindMapInput
}

// getMindMapChildren returns ordered list of children for a node
func (m *model) getMindMapChildren(parentID int) []int {
	if m.mindMapSiblingOrder == nil {
		m.mindMapSiblingOrder = make(map[int][]int)
	}

	if order, ok := m.mindMapSiblingOrder[parentID]; ok {
		// Filter out any deleted nodes
		var valid []int
		for _, id := range order {
			if id < len(m.getCanvas().boxes) {
				if p, exists := m.mindMapParents[id]; exists && p == parentID {
					valid = append(valid, id)
				}
			}
		}
		return valid
	}

	// Build list from mindMapParents
	var children []int
	for childID, pid := range m.mindMapParents {
		if pid == parentID && childID < len(m.getCanvas().boxes) {
			children = append(children, childID)
		}
	}
	// Sort by Y position for consistent ordering
	sort.Slice(children, func(i, j int) bool {
		if children[i] >= len(m.getCanvas().boxes) || children[j] >= len(m.getCanvas().boxes) {
			return children[i] < children[j]
		}
		return m.getCanvas().boxes[children[i]].Y < m.getCanvas().boxes[children[j]].Y
	})
	return children
}

// getMindMapRoots returns all root nodes (nodes with no parent)
func (m *model) getMindMapRoots() []int {
	var roots []int
	for _, box := range m.getCanvas().boxes {
		if _, hasParent := m.mindMapParents[box.ID]; !hasParent {
			roots = append(roots, box.ID)
		}
	}
	// Sort by Y position
	sort.Slice(roots, func(i, j int) bool {
		if roots[i] >= len(m.getCanvas().boxes) || roots[j] >= len(m.getCanvas().boxes) {
			return roots[i] < roots[j]
		}
		return m.getCanvas().boxes[roots[i]].Y < m.getCanvas().boxes[roots[j]].Y
	})
	return roots
}

// deleteMindMapNode deletes a node and all its descendants
func (m *model) deleteMindMapNode(boxID int) {
	if boxID < 0 || boxID >= len(m.getCanvas().boxes) {
		return
	}

	// First delete all children recursively
	m.deleteMindMapChildren(boxID)

	// Remove from parent's sibling order
	if parentID, ok := m.mindMapParents[boxID]; ok {
		if order, exists := m.mindMapSiblingOrder[parentID]; exists {
			var newOrder []int
			for _, id := range order {
				if id != boxID {
					newOrder = append(newOrder, id)
				}
			}
			m.mindMapSiblingOrder[parentID] = newOrder
		}
	}

	// Remove from mindMapParents
	delete(m.mindMapParents, boxID)

	// Delete the box
	m.getCanvas().DeleteBox(boxID)
}

// deleteMindMapChildren deletes all children of a node
func (m *model) deleteMindMapChildren(boxID int) {
	children := m.getMindMapChildren(boxID)
	for _, childID := range children {
		m.deleteMindMapNode(childID)
	}
	// Clear sibling order for this node
	delete(m.mindMapSiblingOrder, boxID)
}

// moveMindMapNodeUp moves a node up among its siblings
func (m *model) moveMindMapNodeUp(boxID int) {
	parentID, hasParent := m.mindMapParents[boxID]
	if !hasParent {
		parentID = -1
	}

	siblings := m.getSiblings(boxID, parentID)
	if len(siblings) <= 1 {
		return
	}

	// Find current position
	currentIdx := -1
	for i, id := range siblings {
		if id == boxID {
			currentIdx = i
			break
		}
	}

	if currentIdx <= 0 {
		return // Already at top
	}

	// Swap with previous sibling
	siblings[currentIdx], siblings[currentIdx-1] = siblings[currentIdx-1], siblings[currentIdx]
	m.mindMapSiblingOrder[parentID] = siblings

	// Update visual positions
	m.relayoutMindMapSiblings(parentID)
}

// moveMindMapNodeDown moves a node down among its siblings
func (m *model) moveMindMapNodeDown(boxID int) {
	parentID, hasParent := m.mindMapParents[boxID]
	if !hasParent {
		parentID = -1
	}

	siblings := m.getSiblings(boxID, parentID)
	if len(siblings) <= 1 {
		return
	}

	// Find current position
	currentIdx := -1
	for i, id := range siblings {
		if id == boxID {
			currentIdx = i
			break
		}
	}

	if currentIdx < 0 || currentIdx >= len(siblings)-1 {
		return // Already at bottom
	}

	// Swap with next sibling
	siblings[currentIdx], siblings[currentIdx+1] = siblings[currentIdx+1], siblings[currentIdx]
	m.mindMapSiblingOrder[parentID] = siblings

	// Update visual positions
	m.relayoutMindMapSiblings(parentID)
}

// getSiblings returns all siblings of a node (including itself)
func (m *model) getSiblings(boxID int, parentID int) []int {
	if m.mindMapSiblingOrder == nil {
		m.mindMapSiblingOrder = make(map[int][]int)
	}

	if order, ok := m.mindMapSiblingOrder[parentID]; ok && len(order) > 0 {
		return order
	}

	// Build from mindMapParents
	var siblings []int
	if parentID == -1 {
		// Root nodes
		siblings = m.getMindMapRoots()
	} else {
		siblings = m.getMindMapChildren(parentID)
	}

	m.mindMapSiblingOrder[parentID] = siblings
	return siblings
}

// getSubtreeHeight calculates the total vertical space needed by a node and all its descendants
func (m *model) getSubtreeHeight(nodeID int) int {
	if nodeID < 0 || nodeID >= len(m.getCanvas().boxes) {
		return 0
	}

	children := m.getMindMapChildren(nodeID)
	if len(children) == 0 {
		// Leaf node - just return its own height
		return m.getCanvas().boxes[nodeID].Height
	}

	// Sum up all children's subtree heights plus spacing
	totalChildrenHeight := 0
	for i, childID := range children {
		totalChildrenHeight += m.getSubtreeHeight(childID)
		if i < len(children)-1 {
			totalChildrenHeight += 1 // spacing between siblings
		}
	}

	// The subtree height is the max of the node's own height and its children's total height
	nodeHeight := m.getCanvas().boxes[nodeID].Height
	if totalChildrenHeight > nodeHeight {
		return totalChildrenHeight
	}
	return nodeHeight
}

// relayoutMindMapTree performs a full tree layout starting from the root
func (m *model) relayoutMindMapTree() {
	// Find all root nodes and layout each tree
	roots := m.getMindMapRoots()
	for _, rootID := range roots {
		m.layoutSubtree(rootID)
	}
	// Update all connections after layout
	m.updateAllMindMapConnections()
}

// layoutSubtree recursively lays out a node and all its descendants
func (m *model) layoutSubtree(nodeID int) {
	if nodeID < 0 || nodeID >= len(m.getCanvas().boxes) {
		return
	}

	children := m.getMindMapChildren(nodeID)
	if len(children) == 0 {
		return
	}

	nodeBox := m.getCanvas().boxes[nodeID]
	nodeCenterY := nodeBox.Y + nodeBox.Height/2

	// Calculate total height needed for all children's subtrees
	subtreeHeights := make([]int, len(children))
	totalHeight := 0
	for i, childID := range children {
		subtreeHeights[i] = m.getSubtreeHeight(childID)
		totalHeight += subtreeHeights[i]
		if i < len(children)-1 {
			totalHeight += 1 // spacing
		}
	}

	// Position children centered around the parent's center Y
	startY := nodeCenterY - totalHeight/2
	currentY := startY

	for i, childID := range children {
		if childID >= len(m.getCanvas().boxes) {
			continue
		}
		childBox := &m.getCanvas().boxes[childID]

		// Position child at the center of its allocated subtree space
		subtreeHeight := subtreeHeights[i]
		childBox.Y = currentY + (subtreeHeight-childBox.Height)/2

		currentY += subtreeHeight + 1

		// Recursively layout this child's subtree
		m.layoutSubtree(childID)
	}
}

// relayoutMindMapSiblings updates layout for siblings and propagates changes up the tree
func (m *model) relayoutMindMapSiblings(parentID int) {
	// Do a full tree relayout to handle all cascading changes
	m.relayoutMindMapTree()
}

// updateAllMindMapConnections updates all mind map connection positions
func (m *model) updateAllMindMapConnections() {
	for parentID := range m.mindMapParents {
		m.updateMindMapConnections(m.mindMapParents[parentID])
	}
	// Also update connections for all nodes that have children
	for nodeID := range m.getCanvas().boxes {
		m.updateMindMapConnections(nodeID)
	}
}

// updateMindMapConnections updates connection positions for a parent's children
func (m *model) updateMindMapConnections(parentID int) {
	if parentID < 0 || parentID >= len(m.getCanvas().boxes) {
		return
	}

	children := m.getMindMapChildren(parentID)
	if len(children) == 0 {
		return
	}

	parentBox := m.getCanvas().boxes[parentID]

	for _, childID := range children {
		if childID >= len(m.getCanvas().boxes) {
			continue
		}
		childBox := m.getCanvas().boxes[childID]

		// Find and update the connection between parent and child
		for i := range m.getCanvas().connections {
			conn := &m.getCanvas().connections[i]
			if conn.FromID == parentID && conn.ToID == childID {
				conn.FromX = parentBox.X + parentBox.Width
				conn.FromY = parentBox.Y + parentBox.Height/2
				conn.ToX = childBox.X - 1
				conn.ToY = childBox.Y + childBox.Height/2
				break
			}
		}
	}
}

// pasteMindMapAsChild pastes yanked node as a child
func (m *model) pasteMindMapAsChild(parentID int) {
	if m.mindMapYankedNode < 0 || m.mindMapYankedNode >= len(m.getCanvas().boxes) {
		return
	}

	sourceBox := m.getCanvas().boxes[m.mindMapYankedNode]
	parentBox := m.getCanvas().boxes[parentID]

	// Calculate position for new node
	children := m.getMindMapChildren(parentID)
	var newY int
	if len(children) > 0 {
		lastChild := children[len(children)-1]
		if lastChild < len(m.getCanvas().boxes) {
			lastBox := m.getCanvas().boxes[lastChild]
			newY = lastBox.Y + lastBox.Height + 1
		} else {
			newY = parentBox.Y
		}
	} else {
		newY = parentBox.Y
	}
	newX := parentBox.X + parentBox.Width + 4

	// Create the new node
	newID := m.getCanvas().AddMindMapNode(newX, newY, sourceBox.GetText())
	m.mindMapParents[newID] = parentID

	// Add to sibling order
	if m.mindMapSiblingOrder == nil {
		m.mindMapSiblingOrder = make(map[int][]int)
	}
	m.mindMapSiblingOrder[parentID] = append(m.mindMapSiblingOrder[parentID], newID)

	// Create connection
	m.createMindMapConnection(parentID, newID)

	// If yanking with children, recursively copy children
	if m.mindMapYankedWithChildren {
		m.copyMindMapChildren(m.mindMapYankedNode, newID)
	}

	// Relayout to center children around parent
	m.relayoutMindMapSiblings(parentID)
}

// pasteMindMapAsSibling pastes yanked node as a sibling
func (m *model) pasteMindMapAsSibling(siblingID int) {
	if m.mindMapYankedNode < 0 || m.mindMapYankedNode >= len(m.getCanvas().boxes) {
		return
	}

	sourceBox := m.getCanvas().boxes[m.mindMapYankedNode]
	siblingBox := m.getCanvas().boxes[siblingID]

	parentID, hasParent := m.mindMapParents[siblingID]
	if !hasParent {
		parentID = -1
	}

	// Position below sibling
	newX := siblingBox.X
	newY := siblingBox.Y + siblingBox.Height + 1

	// Create the new node
	newID := m.getCanvas().AddMindMapNode(newX, newY, sourceBox.GetText())
	if parentID != -1 {
		m.mindMapParents[newID] = parentID
		m.createMindMapConnection(parentID, newID)
	}

	// Add to sibling order after the target sibling
	siblings := m.getSiblings(siblingID, parentID)
	var newOrder []int
	for _, id := range siblings {
		newOrder = append(newOrder, id)
		if id == siblingID {
			newOrder = append(newOrder, newID)
		}
	}
	m.mindMapSiblingOrder[parentID] = newOrder

	// If yanking with children, recursively copy children
	if m.mindMapYankedWithChildren {
		m.copyMindMapChildren(m.mindMapYankedNode, newID)
	}

	// Relayout to center siblings around parent
	if parentID != -1 {
		m.relayoutMindMapSiblings(parentID)
	}
}

// copyMindMapChildren recursively copies children from source to target
func (m *model) copyMindMapChildren(sourceID, targetID int) {
	children := m.getMindMapChildren(sourceID)
	for _, childID := range children {
		if childID >= len(m.getCanvas().boxes) {
			continue
		}
		childBox := m.getCanvas().boxes[childID]
		targetBox := m.getCanvas().boxes[targetID]

		// Position relative to target
		newX := targetBox.X + targetBox.Width + 4
		existingChildren := m.getMindMapChildren(targetID)
		var newY int
		if len(existingChildren) > 0 {
			lastChild := existingChildren[len(existingChildren)-1]
			if lastChild < len(m.getCanvas().boxes) {
				lastBox := m.getCanvas().boxes[lastChild]
				newY = lastBox.Y + lastBox.Height + 1
			} else {
				newY = targetBox.Y
			}
		} else {
			newY = targetBox.Y
		}

		// Create copy
		newChildID := m.getCanvas().AddMindMapNode(newX, newY, childBox.GetText())
		m.mindMapParents[newChildID] = targetID
		m.createMindMapConnection(targetID, newChildID)

		// Recursively copy grandchildren
		m.copyMindMapChildren(childID, newChildID)
	}
}

// createMindMapConnection creates a connection between parent and child
func (m *model) createMindMapConnection(parentID, childID int) {
	if parentID < 0 || parentID >= len(m.getCanvas().boxes) {
		return
	}
	if childID < 0 || childID >= len(m.getCanvas().boxes) {
		return
	}

	parentBox := m.getCanvas().boxes[parentID]
	childBox := m.getCanvas().boxes[childID]

	fromX := parentBox.X + parentBox.Width
	fromY := parentBox.Y + parentBox.Height/2
	toX := childBox.X - 1
	toY := childBox.Y + childBox.Height/2

	conn := Connection{
		FromID: parentID,
		ToID:   childID,
		FromX:  fromX,
		FromY:  fromY,
		ToX:    toX,
		ToY:    toY,
	}
	m.getCanvas().connections = append(m.getCanvas().connections, conn)
}

// goToMindMapRoot moves cursor to the first root node
func (m *model) goToMindMapRoot() {
	roots := m.getMindMapRoots()
	if len(roots) == 0 {
		return
	}

	rootID := roots[0]
	if rootID >= len(m.getCanvas().boxes) {
		return
	}

	box := m.getCanvas().boxes[rootID]
	panX, panY := m.getPanOffset()
	m.cursorX = box.X + box.Width/2 - panX
	m.cursorY = box.Y + box.Height/2 - panY
	m.ensureCursorInBounds()
	m.selectedMindMapNode = rootID
}

// goToFirstSibling moves cursor to the first sibling
func (m *model) goToFirstSibling(boxID int) {
	parentID, hasParent := m.mindMapParents[boxID]
	if !hasParent {
		parentID = -1
	}

	siblings := m.getSiblings(boxID, parentID)
	if len(siblings) == 0 {
		return
	}

	firstID := siblings[0]
	if firstID >= len(m.getCanvas().boxes) {
		return
	}

	box := m.getCanvas().boxes[firstID]
	panX, panY := m.getPanOffset()
	m.cursorX = box.X + box.Width/2 - panX
	m.cursorY = box.Y + box.Height/2 - panY
	m.ensureCursorInBounds()
	m.selectedMindMapNode = firstID
}

// goToLastSibling moves cursor to the last sibling
func (m *model) goToLastSibling(boxID int) {
	parentID, hasParent := m.mindMapParents[boxID]
	if !hasParent {
		parentID = -1
	}

	siblings := m.getSiblings(boxID, parentID)
	if len(siblings) == 0 {
		return
	}

	lastID := siblings[len(siblings)-1]
	if lastID >= len(m.getCanvas().boxes) {
		return
	}

	box := m.getCanvas().boxes[lastID]
	panX, panY := m.getPanOffset()
	m.cursorX = box.X + box.Width/2 - panX
	m.cursorY = box.Y + box.Height/2 - panY
	m.ensureCursorInBounds()
	m.selectedMindMapNode = lastID
}

// addMindMapNodeToOrder adds a new node to the sibling order
func (m *model) addMindMapNodeToOrder(nodeID, parentID, afterSiblingID int) {
	if m.mindMapSiblingOrder == nil {
		m.mindMapSiblingOrder = make(map[int][]int)
	}

	siblings := m.mindMapSiblingOrder[parentID]
	if afterSiblingID == -1 {
		// Add at end
		m.mindMapSiblingOrder[parentID] = append(siblings, nodeID)
	} else {
		// Add after specific sibling
		var newOrder []int
		for _, id := range siblings {
			newOrder = append(newOrder, id)
			if id == afterSiblingID {
				newOrder = append(newOrder, nodeID)
			}
		}
		// If afterSiblingID wasn't found, add at end
		found := false
		for _, id := range newOrder {
			if id == nodeID {
				found = true
				break
			}
		}
		if !found {
			newOrder = append(newOrder, nodeID)
		}
		m.mindMapSiblingOrder[parentID] = newOrder
	}
}
