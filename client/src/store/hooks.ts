/**
 * Typed hooks for Redux with thunk support
 * Use these instead of plain `useDispatch` and `useSelector`
 */
import { useDispatch as useReduxDispatch } from 'react-redux';
import { ThunkDispatch } from 'redux-thunk';
import { AnyAction } from 'redux';
import { RootState } from '@/initialState';

// Export a hook that can be reused to resolve types
export type AppDispatch = ThunkDispatch<RootState, unknown, AnyAction>;

export const useDispatch = () => useReduxDispatch<AppDispatch>();
// For useSelector, just re-export the original to maintain flexibility
export { useSelector } from 'react-redux';
