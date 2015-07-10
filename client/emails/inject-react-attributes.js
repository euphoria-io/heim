// munge in some HTML attributes so React doesn't strip them out
var DOMProperty = require('react/lib/ReactInjection').DOMProperty
DOMProperty.injectDOMPropertyConfig({
  Properties: {
    'xmlns': DOMProperty.MUST_USE_ATTRIBUTE,
    'http-equiv': DOMProperty.MUST_USE_ATTRIBUTE,
    'align': DOMProperty.MUST_USE_ATTRIBUTE,
    'valign': DOMProperty.MUST_USE_ATTRIBUTE,
    'bgcolor': DOMProperty.MUST_USE_ATTRIBUTE,
  }
})
