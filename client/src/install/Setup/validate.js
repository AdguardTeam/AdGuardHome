const validate = (values) => {
    const errors = {};

    if (values.confirm_password !== values.password) {
        errors.confirm_password = 'Password mismatched';
    }

    return errors;
};

export default validate;
