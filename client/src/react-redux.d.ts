import { ThunkDispatch } from 'redux-thunk';
import { AnyAction } from 'redux';
import { RootState } from './initialState';

declare module 'react-redux' {
    // Override useDispatch to return ThunkDispatch by default
    export function useDispatch<
        TDispatch = ThunkDispatch<RootState, unknown, AnyAction>
    >(): TDispatch;
    
    // Extend DefaultRootState to use our RootState
    export interface DefaultRootState extends RootState {}
}
