// CLSF-77b-78: client-side canvas script — split into fragments to satisfy max-lines.
import { SOURCE_DIFF_SCRIPT } from './builderSourceDiff';
import { BUILDER_SCRIPT_BOOT } from './builderCanvasScriptBoot';
import { BUILDER_SCRIPT_CORE } from './builderCanvasScriptCore';
import { BUILDER_SCRIPT_GRAPH } from './builderCanvasScriptGraph';

export const BUILDER_SCRIPT = `${BUILDER_SCRIPT_CORE}${BUILDER_SCRIPT_GRAPH}${SOURCE_DIFF_SCRIPT}${BUILDER_SCRIPT_BOOT}`;
