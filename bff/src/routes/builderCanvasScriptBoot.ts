export const BUILDER_SCRIPT_BOOT = `
        function bindCanvasRoot() {
          var nextRoot = document.getElementById('builder-canvas-root');
          if (!nextRoot || nextRoot === canvasRoot) return;
          if (canvasRoot) {
            canvasRoot.removeEventListener('click', handleCanvasClick);
            canvasRoot.removeEventListener('pointerdown', handlePointerDown);
            canvasRoot.removeEventListener('pointermove', handlePointerMove);
            canvasRoot.removeEventListener('pointerup', handlePointerUp);
            canvasRoot.removeEventListener('pointercancel', handlePointerUp);
          }
          canvasRoot = nextRoot;
          canvasRoot.addEventListener('click', handleCanvasClick);
          canvasRoot.addEventListener('pointerdown', handlePointerDown);
          canvasRoot.addEventListener('pointermove', handlePointerMove);
          canvasRoot.addEventListener('pointerup', handlePointerUp);
          canvasRoot.addEventListener('pointercancel', handlePointerUp);
        }
        function refreshProjectionFromDom() {
          graphShell = document.getElementById('builder-graph');
          var projection = readInitialProjection();
          if (!projection) return;
          activeWorkflowName = inferWorkflowName(projection);
          graphState.loadFromProjection(projection);
          connectingSourceId = null;
          rerenderCanvas();
        }
        function initAuth() { var tokenInput = document.getElementById('builder-token'); var authForm = document.getElementById('builder-auth-form'); tokenInput.value = localStorage.getItem('fenix.builder.bearerToken') || ''; authForm.addEventListener('submit', function (event) { event.preventDefault(); localStorage.setItem('fenix.builder.bearerToken', tokenInput.value.trim()); }); }
        initAuth();
        bindCanvasRoot();
        document.getElementById('builder-add-node').addEventListener('click', handleAddNode);
        document.getElementById('builder-delete-node').addEventListener('click', handleDeleteSelected);
        document.getElementById('builder-connect-node').addEventListener('click', handleConnectSelected);
        document.getElementById('builder-save-graph').addEventListener('click', handleSaveGraph);
        document.getElementById('builder-editor').addEventListener('input', renderSourceDiff);
        document.getElementById('builder-spec-source').addEventListener('input', renderSourceDiff);
        document.addEventListener('keydown', handleKeyDown);
        document.body.addEventListener('htmx:configRequest', function (event) {
          var token = localStorage.getItem('fenix.builder.bearerToken');
          if (token) {
            event.detail.headers.Authorization = 'Bearer ' + token;
          }
        });
        document.body.addEventListener('htmx:afterSwap', function (event) {
          var target = event.target;
          if (!(target instanceof Element)) return;
          if (target.id === 'builder-graph' || target.querySelector('#builder-graph')) {
            bindCanvasRoot();
            refreshProjectionFromDom();
          }
        });
        window.fenixBuilderCanvas = { getSelectedNodeId: getSelectedNodeId, rerenderCanvas: rerenderCanvas };
        rerenderCanvas();
      }());
    <\\/script>`;
