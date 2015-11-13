export function validateEmail(values, strict) {
  let error
  if (!values.email) {
    if (strict) {
      error = 'please enter an email'
    }
  } else if (!/.+@.+/.test(values.email)) {
    error = 'is that an email address?'
  }
  return {email: error}
}

export function validatePassword(values, strict) {
  let error
  if (strict && !values.password) {
    error = 'please enter a password'
  }
  return {password: error}
}

export function validateNewPassword(values, strict) {
  let error
  if (!values.newPassword) {
    if (strict) {
      error = 'please enter a password'
    }
  } else if (values.newPassword.strength !== 'ok') {
    error = 'please choose a stronger password'
  }
  return {newPassword: error}
}

export const minPasswordEntropy = 42
