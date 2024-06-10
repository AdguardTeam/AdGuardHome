import { createStore, applyMiddleware, compose, Reducer } from 'redux';
import thunk from 'redux-thunk';

const middlewares = [thunk];

export default function configureStore<T>(
    reducer: Reducer<T>,
    initialState: any
) {
    const store = createStore(
        reducer,
        initialState,
        compose(applyMiddleware(...middlewares))
    );
    return store;
}
