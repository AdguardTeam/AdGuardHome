import { describe, expect, it } from 'vitest';

import * as actions from 'panel/actions/queryLogs';
import queryLogsReducer from 'panel/reducers/queryLogs';

describe('queryLogs reducer', () => {
    it('does not mark the log as complete when additional loading stops', () => {
        const initialState = queryLogsReducer(undefined, { type: '@@INIT' } as never);

        const nextState = queryLogsReducer(
            {
                ...initialState,
                isEntireLog: false,
                processingAdditionalLogs: true,
                processingGetLogs: true,
            },
            actions.getAdditionalLogsSuccess(),
        );

        expect(nextState.processingAdditionalLogs).toBe(false);
        expect(nextState.processingGetLogs).toBe(false);
        expect(nextState.isEntireLog).toBe(false);
    });
});
