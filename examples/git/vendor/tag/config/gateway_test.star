load("@ytt:assert", "assert")
load("gateway.lib.yml", "gateway")

def test_gateway_when_app_domain_equals_system_domain():
  result = gateway("sys-and-app-domain.com", ["sys-and-app-domain.com"], "sys-namespace", "work-namespace")
  https_servers = https_servers_in_gateway(result)

  # We expect a single https-based entry in the servers list
  assert_equals(1, len(https_servers))

  # We expect a single host name without any namespace restriction
  assert_equals(1, len(https_servers[0]["hosts"]))
  assert_equals("*.sys-and-app-domain.com", https_servers[0]["hosts"][0])
end

def test_gateway_when_app_domain_does_not_equal_system_domain():
  result = gateway("sys-domain.com", ["app-domain.com"], "sys-namespace", "work-namespace")
  https_servers = https_servers_in_gateway(result)

  # We expect two https-based entries in the servers list: one for the system domain, and a different one for the app domain
  assert_equals(2, len(https_servers))

  # We expect the host field of both servers to be restricted by namespace
  assert_equals(1, len(https_servers[0]["hosts"]))
  assert_equals("sys-namespace/*.sys-domain.com", https_servers[0]["hosts"][0])
  assert_equals(1, len(https_servers[1]["hosts"]))
  assert_equals("work-namespace/*.app-domain.com", https_servers[1]["hosts"][0])
end

def test_gateway_when_one_app_domain_equals_system_domain_and_another_does_not():
  result = gateway("sys-and-app-domain.com", ["sys-and-app-domain.com", "app-domain.com"], "sys-namespace", "work-namespace")
  https_servers = https_servers_in_gateway(result)

  # We expect two https-based entries in the servers list: one for the system domain, and a different one for the app domain
  assert_equals(2, len(https_servers))

  # We expect the host field of the the sys-and-app-domain.com server not to be restricted by namespace
  assert_equals(1, len(https_servers[0]["hosts"]))
  assert_equals("*.sys-and-app-domain.com", https_servers[0]["hosts"][0])

  # We expect the host field of the the app-domain.com server to be restricted by namespace
  assert_equals(1, len(https_servers[1]["hosts"]))
  assert_equals("work-namespace/*.app-domain.com", https_servers[1]["hosts"][0])
end

def test_gateway_when_multiple_app_domains():
  result = gateway("sys-domain.com", ["app-domain-1.com", "app-domain-2.com"], "sys-namespace", "work-namespace")
  https_servers = https_servers_in_gateway(result)

  # We expect two https-based entries in the servers list: one for the system domain, and a different one for the app domain
  assert_equals(2, len(https_servers))

  # We expect two hosts, for app-domain-1.com and app-domain-2.com, and for them to be restricted by namespace
  assert_equals(2, len(https_servers[1]["hosts"]))
  assert_equals("work-namespace/*.app-domain-1.com", https_servers[1]["hosts"][0])
  assert_equals("work-namespace/*.app-domain-2.com", https_servers[1]["hosts"][1])
end

def assert_equals(expected, actual):
  if actual != expected:
    assert.fail("Expected %s, but found %s" % (expected, actual))
  end
end


def https_servers_in_gateway(gateway_yaml):
  return [ server for server in gateway_yaml["spec"]["servers"] if server["port"]["protocol"] == 'HTTPS' ]
end

test_gateway_when_app_domain_equals_system_domain()
test_gateway_when_app_domain_does_not_equal_system_domain()
test_gateway_when_one_app_domain_equals_system_domain_and_another_does_not()
test_gateway_when_multiple_app_domains()
