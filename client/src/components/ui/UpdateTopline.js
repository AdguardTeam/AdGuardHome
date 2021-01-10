import React from 'react';
import { Trans } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import Topline from './Topline';
import { getUpdate } from '../../actions';

const UpdateTopline = () => {
    const {
        announcementUrl,
        newVersion,
        canAutoUpdate,
        processingUpdate,
    } = useSelector((state) => state.dashboard, shallowEqual);
    const dispatch = useDispatch();

    const handleUpdate = () => {
        dispatch(getUpdate());
    };

    return <Topline type="info">
        <>
            <Trans
                values={{ version: newVersion }}
                components={[
                    <a href={announcementUrl} target="_blank" rel="noopener noreferrer" key="0">
                        Click here
                    </a>,
                ]}
            >
                update_announcement
            </Trans>
            {canAutoUpdate
            && <button
                type="button"
                className="btn btn-sm btn-primary ml-3"
                onClick={handleUpdate}
                disabled={processingUpdate}
            >
                <Trans>update_now</Trans>
            </button>
            }
        </>
    </Topline>;
};

export default UpdateTopline;
