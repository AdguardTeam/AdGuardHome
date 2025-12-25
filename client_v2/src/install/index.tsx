import React from 'react';
import ReactDOM from 'react-dom';
import { Provider } from 'react-redux';

import '../index.pcss';
import configureStore from '../configureStore';
import reducers from '../reducers/install';
import '../i18n';

import { Setup } from './Setup';
import { InstallState } from '../initialState';
import { Icons } from '../common/ui/Icons';

const store = configureStore<InstallState>(reducers, {});

ReactDOM.render(
    <Provider store={store}>
        <Setup />
        <Icons />
    </Provider>,
    document.getElementById('root'),
);
