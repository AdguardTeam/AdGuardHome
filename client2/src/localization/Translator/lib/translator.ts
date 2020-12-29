import parser from './parser';
import format, { AllowedValues } from './formatter';

const translator = <T>(message: string, values: AllowedValues<T>) => {
    const astMessage = parser(message);
    const formatted = format<T>(astMessage, values);
    return formatted;
};
export default translator;
