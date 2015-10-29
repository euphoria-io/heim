import { DOMProperty } from 'react/lib/ReactInjection'


export default function injectReactAttributes() {
  // munge in some HTML attributes so React doesn't strip them out
  DOMProperty.injectDOMPropertyConfig({
    Properties: {
      'xmlns': DOMProperty.MUST_USE_ATTRIBUTE,
      'align': DOMProperty.MUST_USE_ATTRIBUTE,
      'valign': DOMProperty.MUST_USE_ATTRIBUTE,
      'bgcolor': DOMProperty.MUST_USE_ATTRIBUTE,
    },
  })
}
