import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';
import ReactModal from 'react-modal';

import Form from './Form';

const Modal = (props) => {
    const {
        isModalOpen,
        handleSubmit,
        toggleRewritesModal,
        processingAdd,
        processingDelete,
    } = props;

    return (
        <ReactModal
            className="Modal__Bootstrap modal-dialog modal-dialog-centered"
            closeTimeoutMS={0}
            isOpen={isModalOpen}
            onRequestClose={() => toggleRewritesModal()}
        >
            <div className="modal-content">
                <div className="modal-header">
                    <h4 className="modal-title">
                        <Trans>rewrite_add</Trans>
                    </h4>
                    <button type="button" className="close" onClick={() => toggleRewritesModal()}>
                        <span className="sr-only">Close</span>
                    </button>
                </div>
                <Form
                    onSubmit={handleSubmit}
                    toggleRewritesModal={toggleRewritesModal}
                    processingAdd={processingAdd}
                    processingDelete={processingDelete}
                />
            </div>
        </ReactModal>
    );
};

Modal.propTypes = {
    isModalOpen: PropTypes.bool.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    toggleRewritesModal: PropTypes.func.isRequired,
    processingAdd: PropTypes.bool.isRequired,
    processingDelete: PropTypes.bool.isRequired,
};

export default withNamespaces()(Modal);
