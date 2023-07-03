import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';
import ReactModal from 'react-modal';

import { MODAL_TYPE } from '../../../helpers/constants';
import Form from './Form';

const Modal = (props) => {
    const {
        isModalOpen,
        handleSubmit,
        toggleRewritesModal,
        processingAdd,
        processingDelete,
        modalType,
        currentRewrite,
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
                        {modalType === MODAL_TYPE.EDIT_REWRITE ? (
                            <Trans>rewrite_edit</Trans>
                        ) : (
                            <Trans>rewrite_add</Trans>
                        )}
                    </h4>
                    <button type="button" className="close" onClick={() => toggleRewritesModal()}>
                        <span className="sr-only">Close</span>
                    </button>
                </div>
                <Form
                    initialValues={{ ...currentRewrite }}
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
    modalType: PropTypes.string.isRequired,
    currentRewrite: PropTypes.object,
};

export default withTranslation()(Modal);
