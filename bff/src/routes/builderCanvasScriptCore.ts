export const BUILDER_SCRIPT_CORE = `<script>
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
        var graphShell = document.getElementById('builder-graph');
        var initialProjection = readInitialProjection();
        var activeWorkflowName = inferWorkflowName(initialProjection);
        var graphState = createGraphState();
        var canvasRoot = null;
        var connectingSourceId = null;
        var dragState = null;
        var suppressNextClick = false;
        graphState.loadFromProjection(initialProjection || fixtureGraph);
        function readInitialProjection() {
          if (graphShell) {
            var payload = graphShell.getAttribute('data-projection-payload');
            if (payload) {
              try { return JSON.parse(payload); } catch (_payloadError) {}
            }
          }
          var raw = document.getElementById('builder-initial-projection');
          if (!raw || !raw.textContent) return null;
          try { return JSON.parse(raw.textContent); } catch (_error) { return null; }
        }
        function inferWorkflowName(projection) {
          if (projection && projection.workflow_name) return projection.workflow_name;
          if (graphShell) {
            var shellName = graphShell.getAttribute('data-workflow-name');
            if (shellName) return shellName;
          }
          return fixtureGraph.workflow_name;
        }
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
              var graph = this.toPayload(activeWorkflowName).graph;
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
        function currentGraph() { return graphState.toPayload(activeWorkflowName).graph; }
        function currentPayload() { return graphState.toPayload(activeWorkflowName); }
        function updateProjectionPayload(projection) {
          graphShell = document.getElementById('builder-graph');
          if (!graphShell) return;
          graphShell.setAttribute('data-projection-source', 'api');
          graphShell.setAttribute('data-workflow-name', projection.workflow_name || activeWorkflowName);
          graphShell.setAttribute('data-projection-payload', JSON.stringify(projection));
          var raw = document.getElementById('builder-initial-projection');
          if (raw) raw.textContent = JSON.stringify(projection);
        }
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
          group.appendChild(svgElement('rect', { class: nodeClass(node.kind), x: node.position.x, y: node.position.y, width: NODE_WIDTH, height: NODE_HEIGHT, rx: NODE_RADIUS }));
          group.appendChild(textElement('graph-label', node.position.x + 20, node.position.y + 32, node.kind));
          group.appendChild(textElement('graph-meta', node.position.x + 20, node.position.y + 54, node.label));
          svg.appendChild(group);
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
`;
