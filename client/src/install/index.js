import React from 'react';
import ReactDOM from 'react-dom';
import { Provider } from 'react-redux';

import '../components/App/index.css';
import '../components/ui/ReactTable.css';
import configureStore from '../configureStore';
import reducers from '../reducers/install';
import '../i18n';
import Setup from './Setup';

const store = configureStore(reducers, {}); // set initial state
ReactDOM.render(
    <Provider store={store}>
        <Setup />
    </Provider>,
    document.getElementById('root'),
);
