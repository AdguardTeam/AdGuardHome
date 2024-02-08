import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';
import ReactModal from 'react-modal';

import { MODAL_TYPE } from '../../../helpers/constants';
import Form from './Form';

const getInitialData = ({
    initial, modalType, clientId, clientName,
}) => {
    if (initial && initial.blocked_services) {
        const { blocked_services } = initial;
        const blocked = {};

        blocked_services.forEach((service) => {
            blocked[service] = true;
        });

        return {
            ...initial,
            blocked_services: blocked,
        };
    }

    if (modalType !== MODAL_TYPE.EDIT_CLIENT && clientId) {
        return {
            ...initial,
            name: clientName,
            ids: [clientId],
        };
    }

    return initial;
};

const Modal = ({
    isModalOpen,
    modalType,
    currentClientData,
    handleSubmit,
    handleClose,
    processingAdding,
    processingUpdating,
    tagsOptions,
    clientId,
    t,
}) => {
    const initialData = getInitialData({
        initial: currentClientData,
        modalType,
        clientId,
        clientName: t('client_name', { id: clientId }),
    });

    return (
        <ReactModal
            className="Modal__Bootstrap modal-dialog modal-dialog-centered modal-dialog--clients"
            closeTimeoutMS={0}
            isOpen={isModalOpen}
            onRequestClose={handleClose}
        >
            <div className="modal-content">
                <div className="modal-header">
                    <h4 className="modal-title">
                        {modalType === MODAL_TYPE.EDIT_CLIENT ? (
                            <Trans>client_edit</Trans>
                        ) : (
                            <Trans>client_new</Trans>
                        )}
                    </h4>
                    <button type="button" className="close" onClick={handleClose}>
                        <span className="sr-only">Close</span>
                    </button>
                </div>
                <Form
                    initialValues={{ ...initialData }}
                    onSubmit={handleSubmit}
                    handleClose={handleClose}
                    processingAdding={processingAdding}
                    processingUpdating={processingUpdating}
                    tagsOptions={tagsOptions}
                />
            </div>
        </ReactModal>
    );
};

Modal.propTypes = {
    isModalOpen: PropTypes.bool.isRequired,
    modalType: PropTypes.string.isRequired,
    currentClientData: PropTypes.object.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    handleClose: PropTypes.func.isRequired,
    processingAdding: PropTypes.bool.isRequired,
    processingUpdating: PropTypes.bool.isRequired,
    tagsOptions: PropTypes.array.isRequired,
    t: PropTypes.func.isRequired,
    clientId: PropTypes.string,
};

export default withTranslation()(Modal);
