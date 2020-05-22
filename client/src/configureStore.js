import { createStore, applyMiddleware, compose } from 'redux';
import thunk from 'redux-thunk';

const middlewares = [
    thunk,
];

export default function configureStore(reducer, initialState) {
    /* eslint-disable no-underscore-dangle */
    const store = createStore(reducer, initialState, compose(
        applyMiddleware(...middlewares),
        window.__REDUX_DEVTOOLS_EXTENSION__ ? window.__REDUX_DEVTOOLS_EXTENSION__() : (f) => f,
    ));
    /* eslint-enable */
    return store;
}
