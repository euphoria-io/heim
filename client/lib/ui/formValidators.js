export function validateEmail(values, strict) {
  const name = Object.keys(values)[0]
  const value = values[name]
  let error
  if (!value) {
    if (strict) {
      error = 'please enter an email'
    }
  } else if (!/.+@.+/.test(value)) {
    error = 'is that an email address?'
  }
  return {[name]: error}
}

export function validatePassword(values, strict) {
  const name = Object.keys(values)[0]
  const value = values[name]
  let error
  if (strict) {
    if (!value || !value.text) {
      error = 'please enter a password'
    } else if (value.strength === 'weak') {
      error = 'please choose a stronger password'
    }
  }
  return {[name]: error}
}

export const minPasswordEntropy = 42
