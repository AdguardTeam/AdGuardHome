import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';
import ReactModal from 'react-modal';

import { MODAL_TYPE } from '../../../helpers/constants';
import Form from './Form';

const getInitialData = (initial) => {
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

    return initial;
};

const Modal = (props) => {
    const {
        isModalOpen,
        modalType,
        currentClientData,
        handleSubmit,
        toggleClientModal,
        processingAdding,
        processingUpdating,
        tagsOptions,
    } = props;
    const initialData = getInitialData(currentClientData);

    return (
        <ReactModal
            className="Modal__Bootstrap modal-dialog modal-dialog-centered modal-dialog--clients"
            closeTimeoutMS={0}
            isOpen={isModalOpen}
            onRequestClose={() => toggleClientModal()}
        >
            <div className="modal-content">
                <div className="modal-header">
                    <h4 className="modal-title">
                        {modalType === MODAL_TYPE.EDIT_FILTERS ? (
                            <Trans>client_edit</Trans>
                        ) : (
                            <Trans>client_new</Trans>
                        )}
                    </h4>
                    <button type="button" className="close" onClick={() => toggleClientModal()}>
                        <span className="sr-only">Close</span>
                    </button>
                </div>
                <Form
                    initialValues={{ ...initialData }}
                    onSubmit={handleSubmit}
                    toggleClientModal={toggleClientModal}
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
    toggleClientModal: PropTypes.func.isRequired,
    processingAdding: PropTypes.bool.isRequired,
    processingUpdating: PropTypes.bool.isRequired,
    tagsOptions: PropTypes.array.isRequired,
};

export default withTranslation()(Modal);
