import { merge } from 'webpack-merge';
/* eslint-disable-next-line import/extensions */
import common from './webpack.common.js';

export default merge(common, {
    stats: 'minimal',
    performance: {
        hints: false,
    },
});
