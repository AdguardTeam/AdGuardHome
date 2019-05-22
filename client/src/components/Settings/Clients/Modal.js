import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';
import ReactModal from 'react-modal';

import Form from './Form';

const Modal = (props) => {
    const {
        isModalOpen,
        currentClientData,
        handleSubmit,
        toggleClientModal,
        processingAdding,
        processingUpdating,
    } = props;

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
                        <Trans>client_new</Trans>
                    </h4>
                    <button type="button" className="close" onClick={() => toggleClientModal()}>
                        <span className="sr-only">Close</span>
                    </button>
                </div>
                <Form
                    initialValues={{
                        ...currentClientData,
                    }}
                    onSubmit={handleSubmit}
                    toggleClientModal={toggleClientModal}
                    processingAdding={processingAdding}
                    processingUpdating={processingUpdating}
                />
            </div>
        </ReactModal>
    );
};

Modal.propTypes = {
    isModalOpen: PropTypes.bool.isRequired,
    currentClientData: PropTypes.object.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    toggleClientModal: PropTypes.func.isRequired,
    processingAdding: PropTypes.bool.isRequired,
    processingUpdating: PropTypes.bool.isRequired,
};

export default withNamespaces()(Modal);
