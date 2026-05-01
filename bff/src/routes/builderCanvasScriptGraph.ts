export const BUILDER_SCRIPT_GRAPH = `
        function handlePointerDown(event) {
          if (connectingSourceId) return;
          var shell = event.target.closest('[data-node-id]'); var svg = event.target.closest('svg');
          if (!shell || !svg) return;
          var nodeId = shell.getAttribute('data-node-id'); var node = findNode(nodeId); var point = pointFromEvent(svg, event);
          if (!node || !point) return;
          dragState = { nodeId: nodeId, offsetX: point.x - node.position.x, offsetY: point.y - node.position.y, moved: false, startX: point.x, startY: point.y };
          event.currentTarget.setPointerCapture(event.pointerId);
        }
        function handlePointerMove(event) {
          var svg = event.target.closest('svg') || document.querySelector('#builder-canvas-root svg');
          if (!dragState || !svg) return;
          var point = pointFromEvent(svg, event);
          if (!point) return;
          var dx = point.x - dragState.startX; var dy = point.y - dragState.startY;
          dragState.moved = dragState.moved || Math.sqrt(dx * dx + dy * dy) > 2;
          graphState.updateNodePosition(dragState.nodeId, { x: point.x - dragState.offsetX, y: point.y - dragState.offsetY });
          rerenderCanvas();
        }
        function handlePointerUp(event) { if (!dragState) return; suppressNextClick = dragState.moved; dragState = null; event.currentTarget.releasePointerCapture(event.pointerId); }
        function nextNodePosition() { var graph = currentGraph(); return { x: 80 + graph.nodes.length * 28, y: 320 + graph.nodes.length * 18 }; }
        function handleAddNode() {
          var kind = document.getElementById('builder-add-node-kind').value;
          var input = document.getElementById('builder-add-node-label'); graphState.addNode(kind, input.value.trim() || 'New ' + kind, nextNodePosition()); rerenderCanvas();
        }
        function handleDeleteSelected() { var selected = graphState.getSelectedNodeId(); if (!selected) return; if (connectingSourceId === selected) connectingSourceId = null; graphState.removeNode(selected); rerenderCanvas(); }
        function handleConnectSelected() { if (graphState.getSelectedNodeId()) startConnecting(); }
        function handleKeyDown(event) { if (event.key === 'Escape' && connectingSourceId) cancelConnecting(); }
        function startConnecting() { connectingSourceId = graphState.getSelectedNodeId(); rerenderCanvas(); }
        function cancelConnecting() { connectingSourceId = null; rerenderCanvas(); }
        function tryFinishConnecting(targetId) {
          if (!connectingSourceId || !targetId) return false;
          graphState.addEdge(connectingSourceId, targetId, 'execution'); connectingSourceId = null; rerenderCanvas(); return true;
        }
        function authHeaders() { var headers = { Accept: 'application/json', 'Content-Type': 'application/json' }; var token = localStorage.getItem('fenix.builder.bearerToken'); if (token) headers.Authorization = 'Bearer ' + token; return headers; }
        function setStatus(text) { document.getElementById('builder-preview-status').textContent = text; }
        function renderSaveDiagnostics(items) {
          var list = document.getElementById('builder-diagnostics'); list.replaceChildren();
          if (!items || items.length === 0) { var empty = document.createElement('li'); empty.className = 'diagnostic-empty'; empty.textContent = 'No validation diagnostics for current draft.'; list.appendChild(empty); return; }
          items.forEach(function (item) { var li = document.createElement('li'); li.textContent = (item.code || 'diagnostic') + ': ' + (item.description || item.message || 'Validation diagnostic'); list.appendChild(li); });
        }
        function applySaveProjection(data) {
          if (data.projection) {
            activeWorkflowName = data.projection.workflow_name || activeWorkflowName;
            graphState.loadFromProjection(data.projection);
            updateProjectionPayload(data.projection);
          }
          connectingSourceId = null; setStatus('Graph saved'); rerenderCanvas();
        }
        function handleSaveGraph() {
          var button = document.getElementById('builder-save-graph'); var workflowId = button.getAttribute('data-workflow-id');
          if (!workflowId) return; button.disabled = true; setStatus('Saving graph...');
          fetch(button.getAttribute('data-save-action'), { method: 'POST', headers: authHeaders(), body: JSON.stringify(currentPayload()) }).then(function (response) { return response.json().then(function (data) { return { response: response, data: data }; }); }).then(function (result) { if (result.response.status === 422) { setStatus('Save failed — diagnostics'); renderSaveDiagnostics(result.data.diagnostics); } else if (result.response.ok) applySaveProjection(result.data); else setStatus('Save failed'); }).catch(function () { setStatus('Save failed'); }).finally(function () { button.disabled = false; syncToolbar(); });
        }
        function syncToolbar() { var selected = graphState.getSelectedNodeId(); document.getElementById('builder-delete-node').disabled = !selected; document.getElementById('builder-connect-node').disabled = !selected || Boolean(connectingSourceId); document.getElementById('builder-connect-status').textContent = connectingSourceId ? 'Connecting from ' + connectingSourceId : 'Connect idle'; }
`;
