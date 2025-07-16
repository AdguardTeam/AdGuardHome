import autoprefixer from 'autoprefixer';
import postcssImport from 'postcss-import';
import postcssCustomMedia from 'postcss-custom-media';
import postcssNested from 'postcss-nested';

export default {
  plugins: [
    postcssImport(),
    postcssCustomMedia(),
    postcssNested(),
    autoprefixer({ overrideBrowserslist: ['last 2 versions'] }),
  ],
};
