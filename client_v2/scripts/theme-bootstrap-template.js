import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const DIR_PATH = path.dirname(fileURLToPath(import.meta.url));

// The single copy of the pre-paint theme script, inlined into every entry point.
// Read when the webpack config loads, so restart the dev server after editing it.
export const THEME_BOOTSTRAP_PATH = path.resolve(DIR_PATH, '../public/theme-bootstrap.js');

export const THEME_BOOTSTRAP_SOURCE = fs.readFileSync(THEME_BOOTSTRAP_PATH, 'utf8').trim();

// Template parameter each entry point substitutes with the script tag.
export const THEME_BOOTSTRAP_PARAMETER = 'themeBootstrapScript';

const BODY_INDENT = '            ';
const TAG_INDENT = '        ';

export const THEME_BOOTSTRAP_TAG = [
    '<script>',
    ...THEME_BOOTSTRAP_SOURCE.split('\n').map((line) => (line ? `${BODY_INDENT}${line}` : line)),
    `${TAG_INDENT}</script>`,
].join('\n');

// `templateParameters` shared by all HTML entry points.
export const THEME_BOOTSTRAP_TEMPLATE_PARAMETERS = {
    [THEME_BOOTSTRAP_PARAMETER]: THEME_BOOTSTRAP_TAG,
};
