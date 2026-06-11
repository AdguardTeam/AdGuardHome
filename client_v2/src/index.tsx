import React from 'react';
import { createRoot } from 'react-dom/client';
import { Provider } from 'react-redux';
import configureStore from './configureStore';
import reducers from './reducers';

import App from './components/App';
import { RootState, initialState } from './initialState';

import './index.pcss';

const store = configureStore<RootState>(reducers, initialState);

const root = createRoot(document.getElementById('root')!);
root.render(
    <Provider store={store}>
        <App />
    </Provider>,
);
