import React from 'react';
import ReactDOM from 'react-dom';
import { Provider } from 'react-redux';

import '../index.pcss';
import configureStore from '../configureStore';
import reducers from '../reducers/login';
import '../i18n';

import { ForgotPassword } from './ForgotPassword';
import { LoginState } from '../initialState';

const store = configureStore<LoginState>(reducers, {});

ReactDOM.render(
    <Provider store={store}>
        <ForgotPassword />
    </Provider>,
    document.getElementById('root'),
);
