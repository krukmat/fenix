// CLSF-78: browser-side generated source preview and simple line diff for visual authoring.
export const SOURCE_DIFF_SCRIPT = `
        function generateDslSource(graph) {
          var workflow = graph.nodes.find(function (n) { return n.kind === 'workflow'; });
          var trigger = graph.nodes.find(function (n) { return n.kind === 'trigger'; });
          var actions = graph.nodes.filter(function (n) { return n.kind === 'action' || n.kind === 'decision'; });
          var lines = ['WORKFLOW ' + (workflow ? workflow.label : graph.workflow_name || 'visual_workflow')];
          lines.push('ON ' + (trigger ? trigger.label : 'visual.trigger'));
          actions.forEach(function (node) { lines.push(actionLine(node)); });
          return lines.join('\\n');
        }
        function actionLine(node) {
          if (node.kind === 'decision') return 'IF ' + node.label + ' THEN';
          if (node.label.toLowerCase().indexOf('notify') !== -1) return 'NOTIFY ' + node.label;
          return 'SET ' + node.label;
        }
        function generateSpecSource(graph) {
          var governance = graph.nodes.filter(function (n) { return ['grounds', 'permit', 'delegate', 'invariant', 'budget'].indexOf(n.kind) !== -1; });
          if (governance.length === 0) return '';
          var lines = ['CARTA ' + (graph.workflow_name || 'visual_workflow'), 'AGENT visual_agent'];
          governance.forEach(function (node) { lines.push('  ' + specLine(node)); });
          return lines.join('\\n');
        }
        function specLine(node) {
          if (node.kind === 'grounds') return 'GROUNDS ' + node.label;
          if (node.kind === 'delegate') return 'DELEGATE TO HUMAN ' + node.label;
          if (node.kind === 'invariant') return 'INVARIANT ' + node.label;
          if (node.kind === 'budget') return 'BUDGET ' + node.label;
          return 'PERMIT ' + node.label;
        }
        function renderSourceDiff() {
          var target = document.getElementById('builder-source-diff'); var summary = document.getElementById('builder-source-diff-summary');
          if (!target || !summary) return;
          var generated = graphState.generatedSources();
          var diff = buildSourceDiff(currentEditorSource(), generated.dsl_source + '\\n\\n' + generated.spec_source);
          target.replaceChildren(); diff.lines.forEach(function (line) { target.appendChild(diffLineElement(line)); });
          summary.textContent = diff.changed + ' changed line(s) between persisted text and generated visual source.';
        }
        function currentEditorSource() { return document.getElementById('builder-editor').value + '\\n\\n' + document.getElementById('builder-spec-source').value; }
        function buildSourceDiff(previous, generated) {
          var left = previous.split('\\n'); var right = generated.split('\\n'); var count = Math.max(left.length, right.length);
          var rows = []; var changed = 0;
          for (var i = 0; i < count; i += 1) {
            var before = left[i] || ''; var after = right[i] || ''; var status = lineStatus(before, after, i, left.length, right.length);
            if (status !== 'same') changed += 1;
            rows.push({ number: i + 1, status: status, before: before, after: after });
          }
          return { changed: changed, lines: rows };
        }
        function lineStatus(before, after, index, leftCount, rightCount) {
          if (before === after) return 'same';
          if (index >= leftCount) return 'added';
          if (index >= rightCount) return 'removed';
          return before ? 'changed' : 'added';
        }
        function diffLineElement(line) {
          var row = document.createElement('div'); row.className = 'source-diff-line ' + line.status;
          var label = document.createElement('span'); label.textContent = line.status + ' L' + line.number;
          var body = document.createElement('span'); body.textContent = line.status === 'removed' ? line.before : line.after;
          row.appendChild(label); row.appendChild(body); return row;
        }`;
