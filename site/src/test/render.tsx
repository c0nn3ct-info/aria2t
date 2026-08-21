// RTL render for the site. The site's i18n is a module-level dictionary (no
// provider), so this is a thin re-export plus the helpers every suite wants.
export { act, fireEvent, render, screen, waitFor, within, renderHook } from '@testing-library/react';
export { default as userEvent } from '@testing-library/user-event';
