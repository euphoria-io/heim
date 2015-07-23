// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Math/sign
Math.sign = Math.sign || function(x) {
  x = +x  // convert to a number
  if (x === 0 || isNaN(x)) {
    return x
  }
  return x > 0 ? 1 : -1
}

