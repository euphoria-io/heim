export default function isTextInput(el) {
  const isTextAreaEl = el.nodeName === 'TEXTAREA'
  const isTextInputEl = el.nodeName === 'INPUT' && (el.type === 'text' || el.type === 'password')
  return isTextAreaEl || isTextInputEl
}
