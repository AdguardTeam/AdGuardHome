import React from 'react';
import { createRoot } from 'react-dom/client';
import { Provider } from 'react-redux';

import '../index.pcss';
import configureStore from '../configureStore';
import reducers from '../reducers/login';

import { Login } from './Login';
import { LoginState } from '../initialState';

const store = configureStore<LoginState>(reducers, {});

const root = createRoot(document.getElementById('root')!);
root.render(
    <Provider store={store}>
        <Login />
    </Provider>,
);
