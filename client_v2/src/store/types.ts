import type { AnyAction } from 'redux';
import type { ThunkAction, ThunkDispatch } from 'redux-thunk';

import type { RootState } from 'panel/initialState';

export type AppDispatch = ThunkDispatch<RootState, void, AnyAction>;
export type AppGetState = () => RootState;
export type AppThunk<ReturnType = void> = ThunkAction<ReturnType, RootState, void, AnyAction>;
