import { merge } from 'webpack-merge';
import common from './webpack.common.js';

export default merge(common, {
    stats: 'minimal',
    performance: {
        hints: false,
    },
});
