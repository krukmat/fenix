// CLSF-77b-78: client-side SVG canvas render, visual edits, save, and generated-source diff.
import { SOURCE_DIFF_SCRIPT } from './builderSourceDiff';
export const GRAPH_AUTHORING_STYLES = `.graph-toolbar { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; padding: 12px 14px; border-bottom: 1px solid var(--line); background: #fbfcfe; } .graph-toolbar input, .graph-toolbar select { height: 34px; border: 1px solid var(--line); border-radius: 6px; padding: 0 10px; color: var(--text); background: #ffffff; } .graph-toolbar button { height: 34px; border: 0; border-radius: 6px; padding: 0 10px; background: var(--accent); color: #ffffff; font-weight: 700; cursor: pointer; } .graph-toolbar button:disabled { cursor: not-allowed; opacity: 0.48; } .graph-node-shell[data-selected="true"] .graph-node { stroke-width: 4; } .graph-node-shell[data-connecting-source="true"] .graph-node { stroke-dasharray: 6 4; } .source-diff { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; padding: 12px 14px; border-top: 1px solid var(--line); background: #fbfcfe; } .source-diff pre { min-height: 120px; margin: 0; padding: 10px; overflow: auto; border: 1px solid var(--line); border-radius: 6px; color: var(--text); background: #ffffff; font: 12px/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }`;
export const GRAPH_AUTHORING_CONTROLS = `<div class="graph-toolbar" aria-label="Graph authoring controls"><select id="builder-add-node-kind" aria-label="Node kind"><option value="workflow">workflow</option><option value="trigger">trigger</option><option value="action">action</option><option value="decision">decision</option><option value="grounds">grounds</option><option value="permit">permit</option><option value="delegate">delegate</option><option value="invariant">invariant</option><option value="budget">budget</option></select><input id="builder-add-node-label" type="text" placeholder="Node label" aria-label="Node label"><button id="builder-add-node" type="button">Add</button><button id="builder-delete-node" type="button" disabled>Delete</button><button id="builder-connect-node" type="button" disabled>Connect</button><button id="builder-save-graph" type="button" data-workflow-id="sales_followup" data-save-action="/bff/builder/visual-authoring/sales_followup">Save graph</button><span class="preview-status" id="builder-connect-status">Connect idle</span></div>`;
export const GENERATED_SOURCE_DIFF = `<div class="source-diff" aria-label="Generated source diff"><pre id="builder-generated-dsl"></pre><pre id="builder-generated-spec"></pre></div>`;
export const GRAPH_CANVAS_PLACEHOLDER = `<div id="builder-canvas-root" role="img" aria-label="Dynamic workflow graph canvas"></div>`;
export const BUILDER_SCRIPT = `<script>
      (function () {
        var SVG_NS = 'http://www.w3.org/2000/svg';
        var NODE_WIDTH = 170;
        var NODE_HEIGHT = 72;
        var NODE_RADIUS = '8';
        var fixtureGraph = {
          workflow_name: 'sales_followup',
          nodes: [{ id: 'vnode-workflow-1', kind: 'workflow', label: 'sales_followup', position: { x: 40, y: 40 } }, { id: 'vnode-trigger-1', kind: 'trigger', label: 'deal.updated', position: { x: 250, y: 40 } }, { id: 'vnode-action-1', kind: 'action', label: 'notify owner', position: { x: 470, y: 40 } }, { id: 'vnode-grounds-1', kind: 'grounds', label: 'permit + grounds', position: { x: 242, y: 215 } }],
          edges: [{ id: 'vedge-1', from: 'vnode-workflow-1', to: 'vnode-trigger-1', connection_type: 'execution' }, { id: 'vedge-2', from: 'vnode-trigger-1', to: 'vnode-action-1', connection_type: 'execution' }, { id: 'vedge-3', from: 'vnode-trigger-1', to: 'vnode-grounds-1', connection_type: 'execution' }]
        };
        var graphState = createGraphState();
        var connectingSourceId = null;
        var dragState = null;
        var suppressNextClick = false;
        graphState.loadFromProjection(fixtureGraph);
        function createGraphState() {
          var nodes = [];
          var edges = [];
          var kindCounters = {};
          var edgeCounter = 0;
          var selectedNodeId = null;
          return {
            loadFromProjection: function (projection) {
              nodes = projection.nodes.map(function (node) { return { id: node.id, kind: node.kind, label: node.label, position: node.position }; });
              edges = projection.edges.map(function (edge) { return { id: edge.id, from: edge.from, to: edge.to, connection_type: edge.connection_type || 'execution' }; });
              kindCounters = {};
              nodes.forEach(function (node) { kindCounters[node.kind] = Math.max(kindCounters[node.kind] || 0, ordinalFromId(node.id, node.kind)); });
              edgeCounter = edges.length;
            },
            addNode: function (kind, label, position) {
              kindCounters[kind] = (kindCounters[kind] || 0) + 1;
              var id = 'vnode-' + kind + '-' + kindCounters[kind];
              nodes.push({ id: id, kind: kind, label: label, position: position });
              selectedNodeId = id; return id;
            },
            removeNode: function (id) {
              nodes = nodes.filter(function (node) { return node.id !== id; });
              edges = edges.filter(function (edge) { return edge.from !== id && edge.to !== id; });
              if (selectedNodeId === id) selectedNodeId = null;
            },
            addEdge: function (from, to, connectionType) {
              if (from === to) return null;
              var existing = edges.find(function (edge) { return edge.from === from && edge.to === to; });
              if (existing) return existing.id;
              edgeCounter += 1; var id = 'vedge-' + edgeCounter; edges.push({ id: id, from: from, to: to, connection_type: connectionType }); return id;
            },
            generatedSources: function () {
              var graph = this.toPayload(fixtureGraph.workflow_name).graph;
              return { dsl_source: generateDslSource(graph), spec_source: generateSpecSource(graph) };
            },
            toPayload: function (workflowName) {
              return { graph: { workflow_name: workflowName, nodes: nodes.slice(), edges: edges.slice() } };
            },
            updateNodePosition: function (id, position) {
              nodes.forEach(function (node) { if (node.id === id) node.position = position; });
            },
            getSelectedNodeId: function () { return selectedNodeId; },
            setSelectedNodeId: function (id) { selectedNodeId = id; }
          };
        }
        function currentGraph() { return graphState.toPayload(fixtureGraph.workflow_name).graph; }
        function currentPayload() { return graphState.toPayload(fixtureGraph.workflow_name); }
        function ordinalFromId(id, kind) { var match = new RegExp('^vnode-' + kind + '-(\\\\d+)$').exec(id); return match ? Number(match[1]) : 0; }
        function findNode(id) { return currentGraph().nodes.find(function (node) { return node.id === id; }) || null; }
        function svgElement(name, attrs) { var element = document.createElementNS(SVG_NS, name); Object.keys(attrs).forEach(function (key) { element.setAttribute(key, String(attrs[key])); }); return element; }
        function textElement(className, x, y, value) { var text = svgElement('text', { class: className, x: x, y: y }); text.textContent = value; return text; }
        function metadataElement(name, id, value) { var element = svgElement(name, { id: id }); element.textContent = value; return element; }
        function nodeClass(kind) { if (kind === 'action') return 'graph-node action'; if (['grounds', 'permit', 'delegate', 'invariant', 'budget'].indexOf(kind) !== -1) return 'graph-node governance'; return 'graph-node'; }
        function nodeCenter(node) { return { x: node.position.x + NODE_WIDTH / 2, y: node.position.y + NODE_HEIGHT / 2 }; }
        function appendDefs(svg) { var defs = svgElement('defs', {}); var marker = svgElement('marker', { id: 'arrowhead', markerWidth: 10, markerHeight: 8, refX: 9, refY: 4, orient: 'auto' }); marker.appendChild(svgElement('path', { d: 'M0,0 L10,4 L0,8 Z', fill: '#8590a3' })); defs.appendChild(marker); svg.appendChild(defs); }
        function appendEdge(svg, edge, nodeIndex) {
          var from = nodeIndex[edge.from]; var to = nodeIndex[edge.to];
          if (!from || !to) return;
          var start = nodeCenter(from); var end = nodeCenter(to);
          svg.appendChild(svgElement('line', { class: 'graph-edge', x1: start.x, y1: start.y, x2: end.x, y2: end.y, 'data-edge-id': edge.id, 'data-from-node-id': edge.from, 'data-to-node-id': edge.to }));
        }
        function appendNode(svg, node) {
          var group = svgElement('g', { class: 'graph-node-shell', 'data-node-id': node.id, 'data-node-kind': node.kind, 'data-selected': graphState.getSelectedNodeId() === node.id, 'data-connecting-source': connectingSourceId === node.id });
          group.appendChild(svgElement('rect', { class: nodeClass(node.kind), x: node.position.x, y: node.position.y, width: NODE_WIDTH, height: NODE_HEIGHT, rx: NODE_RADIUS })); group.appendChild(textElement('graph-label', node.position.x + 20, node.position.y + 32, node.kind)); group.appendChild(textElement('graph-meta', node.position.x + 20, node.position.y + 54, node.label)); svg.appendChild(group);
        }
        function buildSvg(graph) {
          var width = Math.max.apply(null, graph.nodes.map(function (node) { return node.position.x; })) + NODE_WIDTH + 40;
          var height = Math.max.apply(null, graph.nodes.map(function (node) { return node.position.y; })) + NODE_HEIGHT + 40;
          var svg = svgElement('svg', { class: 'graph-canvas', viewBox: '0 0 ' + width + ' ' + height, role: 'img', 'aria-labelledby': 'builder-graph-title builder-graph-desc' });
          svg.appendChild(metadataElement('title', 'builder-graph-title', 'Dynamic workflow graph canvas'));
          svg.appendChild(metadataElement('desc', 'builder-graph-desc', 'Client-side graph rendered from the visual authoring state model.'));
          appendDefs(svg);
          var nodeIndex = {};
          graph.nodes.forEach(function (node) { nodeIndex[node.id] = node; });
          graph.edges.forEach(function (edge) { appendEdge(svg, edge, nodeIndex); });
          graph.nodes.forEach(function (node) { appendNode(svg, node); });
          return svg;
        }
        function rerenderCanvas() {
          var root = document.getElementById('builder-canvas-root'); if (!root) return;
          var graph = currentGraph();
          if (graph.nodes.length === 0) { root.textContent = 'No graph nodes available.'; syncToolbar(); return; }
          root.replaceChildren(buildSvg(graph)); syncToolbar(); renderSourceDiff();
        }
        function getSelectedNodeId() { return graphState.getSelectedNodeId(); }
        function pointFromEvent(svg, event) {
          var matrix = svg.getScreenCTM();
          if (!matrix) return null;
          var point = svg.createSVGPoint(); point.x = event.clientX; point.y = event.clientY;
          return point.matrixTransform(matrix.inverse());
        }
        function handleCanvasClick(event) {
          if (suppressNextClick) {
            suppressNextClick = false;
            return;
          }
          var node = event.target.closest('[data-node-id]');
          if (connectingSourceId) {
            if (!node) cancelConnecting(); else tryFinishConnecting(node.getAttribute('data-node-id'));
            return;
          }
          graphState.setSelectedNodeId(node ? node.getAttribute('data-node-id') : null);
          rerenderCanvas();
        }
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
        function applySaveProjection(data) { if (data.projection) graphState.loadFromProjection(data.projection); connectingSourceId = null; setStatus('Graph saved'); rerenderCanvas(); }
        function handleSaveGraph() {
          var button = document.getElementById('builder-save-graph'); var workflowId = button.getAttribute('data-workflow-id');
          if (!workflowId) return; button.disabled = true; setStatus('Saving graph...');
          fetch(button.getAttribute('data-save-action'), { method: 'POST', headers: authHeaders(), body: JSON.stringify(currentPayload()) }).then(function (response) { return response.json().then(function (data) { return { response: response, data: data }; }); }).then(function (result) { if (result.response.status === 422) { setStatus('Save failed — diagnostics'); renderSaveDiagnostics(result.data.diagnostics); } else if (result.response.ok) applySaveProjection(result.data); else setStatus('Save failed'); }).catch(function () { setStatus('Save failed'); }).finally(function () { button.disabled = false; syncToolbar(); });
        }
        function syncToolbar() { var selected = graphState.getSelectedNodeId(); document.getElementById('builder-delete-node').disabled = !selected; document.getElementById('builder-connect-node').disabled = !selected || Boolean(connectingSourceId); document.getElementById('builder-connect-status').textContent = connectingSourceId ? 'Connecting from ' + connectingSourceId : 'Connect idle'; }
${SOURCE_DIFF_SCRIPT}
        function initAuth() { var tokenInput = document.getElementById('builder-token'); var authForm = document.getElementById('builder-auth-form'); tokenInput.value = localStorage.getItem('fenix.builder.bearerToken') || ''; authForm.addEventListener('submit', function (event) { event.preventDefault(); localStorage.setItem('fenix.builder.bearerToken', tokenInput.value.trim()); }); }
        initAuth();
        var canvasRoot = document.getElementById('builder-canvas-root');
        document.getElementById('builder-add-node').addEventListener('click', handleAddNode);
        document.getElementById('builder-delete-node').addEventListener('click', handleDeleteSelected);
        document.getElementById('builder-connect-node').addEventListener('click', handleConnectSelected);
        document.getElementById('builder-save-graph').addEventListener('click', handleSaveGraph);
        document.getElementById('builder-editor').addEventListener('input', renderSourceDiff);
        document.getElementById('builder-spec-source').addEventListener('input', renderSourceDiff);
        canvasRoot.addEventListener('click', handleCanvasClick);
        canvasRoot.addEventListener('pointerdown', handlePointerDown);
        canvasRoot.addEventListener('pointermove', handlePointerMove);
        canvasRoot.addEventListener('pointerup', handlePointerUp);
        canvasRoot.addEventListener('pointercancel', handlePointerUp);
        document.addEventListener('keydown', handleKeyDown);
        document.body.addEventListener('htmx:configRequest', function (event) {
          var token = localStorage.getItem('fenix.builder.bearerToken');
          if (token) {
            event.detail.headers.Authorization = 'Bearer ' + token;
          }
        });
        window.fenixBuilderCanvas = { getSelectedNodeId: getSelectedNodeId, rerenderCanvas: rerenderCanvas };
        rerenderCanvas();
      }());
    <\/script>`;
