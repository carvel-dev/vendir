# filename starts with '0-' to make sure this file gets
# processed first, consequently forcing version check run first

load("@ytt:version", "version")

version.require_at_least("0.26.0")
