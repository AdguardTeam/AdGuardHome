import type { ReactNode } from 'react';
import { handleActions } from 'redux-actions';
import { nanoid } from 'nanoid';

import { addErrorToast, addNoticeToast, addSuccessToast } from '../actions/toasts';
import { removeToast } from '../actions';
import { TOAST_TYPES } from '../helpers/constants';

type SuccessToastPayload =
    | string
    | {
          message: ReactNode;
          actionLabel?: string;
          undoId?: string;
          code?: string;
      };

const toasts = handleActions(
    {
        [addErrorToast.toString()]: (state: any, { payload }: any) => {
            const message = payload.error.toString();
            console.error(payload.error);

            const errorToast: any = {
                id: nanoid(),
                message,
                options: payload.options,
                type: TOAST_TYPES.ERROR,
            };

            if (payload.action) {
                errorToast.action = payload.action;
            }

            const newState = { ...state, notices: [...state.notices, errorToast] };
            return newState;
        },
        [addSuccessToast.toString()]: (state: any, { payload }: any) => {
            const successPayload = payload as SuccessToastPayload;
            const message =
                typeof successPayload === 'string' ? successPayload : successPayload.message;

            const successToast = {
                id: nanoid(),
                message,
                actionLabel:
                    typeof successPayload === 'string' ? undefined : successPayload.actionLabel,
                undoId: typeof successPayload === 'string' ? undefined : successPayload.undoId,
                code: typeof successPayload === 'string' ? undefined : successPayload.code,
                type: TOAST_TYPES.SUCCESS,
            };

            const newState = { ...state, notices: [...state.notices, successToast] };
            return newState;
        },
        [addNoticeToast.toString()]: (state: any, { payload }: any) => {
            const noticeToast = {
                id: nanoid(),
                message: payload.error.toString(),
                options: payload.options,
                type: TOAST_TYPES.NOTICE,
            };

            const newState = { ...state, notices: [...state.notices, noticeToast] };
            return newState;
        },
        [removeToast.toString()]: (state: any, { payload }: any) => {
            const filtered = state.notices.filter((notice: any) => notice.id !== payload);
            const newState = { ...state, notices: filtered };
            return newState;
        },
    },
    { notices: [] },
);

export default toasts;
