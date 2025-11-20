import { useEffect, RefObject } from 'react';

/**
 * Custom hook to detect clicks outside of a referenced element
 * @param ref - React ref object pointing to the element
 * @param handler - Callback function to execute when click outside is detected
 */
export const useClickOutside = <T extends HTMLElement = HTMLElement>(
    ref: RefObject<T>,
    handler: (event: MouseEvent | TouchEvent) => void,
) => {
    useEffect(() => {
        const listener = (event: MouseEvent | TouchEvent) => {
            // Do nothing if clicking ref's element or descendent elements
            if (!ref.current || ref.current.contains(event.target as Node)) {
                return;
            }
            handler(event);
        };

        document.addEventListener('mousedown', listener);
        document.addEventListener('touchstart', listener);

        return () => {
            document.removeEventListener('mousedown', listener);
            document.removeEventListener('touchstart', listener);
        };
    }, [ref, handler]);
};
