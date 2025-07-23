import autoprefixer from 'autoprefixer';
import postcssImport from 'postcss-import';
import postcssNested from 'postcss-nested';

export default {
    plugins: [postcssImport(), postcssNested(), autoprefixer({ overrideBrowserslist: ['last 2 versions'] })],
};
