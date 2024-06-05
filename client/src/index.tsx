import React from 'react';
import ReactDOM from 'react-dom';
import { Provider } from 'react-redux';
import configureStore from './configureStore';
import reducers from './reducers';

import App from './components/App';
import './components/App/index.css';
import './i18n';
import { RootState, initialState } from './initialState';

const store = configureStore<RootState>(reducers, initialState);

ReactDOM.render(
    <Provider store={store}>
        <App />
    </Provider>,
    document.getElementById('root'),
);
