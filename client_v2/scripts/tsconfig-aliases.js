import fs from 'node:fs';
import path from 'node:path';

// The only reader of `compilerOptions.paths`, shared by webpack.common.js,
// .storybook/main.ts and vitest.config.ts, so a new alias only has to be
// registered in tsconfig.json.
export function getAliasesFromTsconfig(tsconfigPath, basePath) {
    const tsConfig = JSON.parse(fs.readFileSync(tsconfigPath, 'utf8'));
    const aliases = {};

    if (tsConfig.compilerOptions?.paths) {
        Object.entries(tsConfig.compilerOptions.paths).forEach(([alias, targetArr]) => {
            if (alias.endsWith('/*')) {
                const aliasName = alias.replace('/*', '');
                const target = targetArr[0].replace('/*', '');
                aliases[aliasName] = path.resolve(basePath, target);
            } else {
                aliases[alias] = path.resolve(basePath, targetArr[0]);
            }
        });
    }

    return aliases;
}
