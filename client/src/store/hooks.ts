/**
 * Augment react-redux to support redux-thunk actions
 * This fixes TypeScript errors when dispatching thunk actions with react-redux v8
 */
import 'react-redux';
import { ThunkDispatch } from 'redux-thunk';
import { AnyAction } from 'redux';
import { RootState } from '../initialState';

declare module 'react-redux' {
    export interface DefaultRootState extends RootState {}
    
    // Augment the Dispatch type to include thunk actions
    export type Dispatch<A extends AnyAction = AnyAction> = ThunkDispatch<RootState, unknown, A>;
}
