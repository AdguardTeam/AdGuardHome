import React from 'react';
import ReactDOM from 'react-dom';
import { Provider } from 'react-redux';

import '../components/App/index.css';
import '../components/ui/ReactTable.css';
import configureStore from '../configureStore';
import reducers from '../reducers/login';
import '../i18n';
import Login from './Login';

const store = configureStore(reducers, {}); // set initial state
ReactDOM.render(
    <Provider store={store}>
        <Login />
    </Provider>,
    document.getElementById('root'),
);
