import { ThunkDispatch } from 'redux-thunk';
import { AnyAction } from 'redux';

declare module 'react-redux' {
    // Override useDispatch to return ThunkDispatch by default
    export function useDispatch<
        TDispatch = ThunkDispatch<any, unknown, AnyAction>
    >(): TDispatch;
}
