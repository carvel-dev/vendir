load("@ytt:assert", "assert")

def must_exist(data_values, data_value_key):
  value = data_values
  keys = data_value_key.split(".")
  for key in keys:
    value = getattr(value, key)
  end
  if len(value) == 0:
   assert.fail(data_value_key + " cannot be empty")
  end
  return value
end
