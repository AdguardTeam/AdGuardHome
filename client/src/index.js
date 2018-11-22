import React from 'react';
import ReactDOM from 'react-dom';
import { Provider } from 'react-redux';
import './components/App/index.css';
import App from './containers/App';
import configureStore from './configureStore';
import reducers from './reducers';
import './i18n';

const store = configureStore(reducers, {}); // set initial state
ReactDOM.render(
    <Provider store={store}>
        <App />
    </Provider>,
    document.getElementById('root'),
);
