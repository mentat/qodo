// Register happy-dom globals (document, window, …) BEFORE any module that
// touches the DOM is imported. This file must contain no testing-library
// imports — ESM evaluates imports first, so registration has to land here.
import { GlobalRegistrator } from '@happy-dom/global-registrator';

GlobalRegistrator.register();
