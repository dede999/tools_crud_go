# Healthcheck
GET http://localhost:8080/health
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"

# 1. Create a new device
POST http://localhost:8080/devices
Content-Type: application/json
{
  "name": "MacBook Pro M3",
  "brand": "Apple",
  "state": "available"
}
HTTP 201
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" isString
jsonpath "$.name" == "MacBook Pro M3"
jsonpath "$.brand" == "Apple"
jsonpath "$.state" == "available"
jsonpath "$.created_at" isString
[Captures]
device_id: jsonpath "$.id"

# 2. Fetch the created device by ID
GET http://localhost:8080/devices/{{device_id}}
HTTP 200
[Asserts]
jsonpath "$.id" == "{{device_id}}"
jsonpath "$.name" == "MacBook Pro M3"
jsonpath "$.brand" == "Apple"
jsonpath "$.state" == "available"

# 3. Query non-existent device returns 404
GET http://localhost:8080/devices/00000000-0000-0000-0000-000000000000
HTTP 404
[Asserts]
jsonpath "$.error" == "device not found"

# 4. Fetch devices by brand filter
GET http://localhost:8080/devices?brand=Apple
HTTP 200
[Asserts]
jsonpath "$[0].brand" == "Apple"

# 5. Fetch devices by state filter
GET http://localhost:8080/devices?state=available
HTTP 200
[Asserts]
jsonpath "$[0].state" == "available"

# 6. Partially update device to 'in-use' state
PATCH http://localhost:8080/devices/{{device_id}}
Content-Type: application/json
{
  "state": "in-use"
}
HTTP 200
[Asserts]
jsonpath "$.id" == "{{device_id}}"
jsonpath "$.state" == "in-use"

# 7. Domain Validation: Name/Brand CANNOT be updated if device is in-use (Expects 409 Conflict)
PATCH http://localhost:8080/devices/{{device_id}}
Content-Type: application/json
{
  "name": "Hacked Name"
}
HTTP 409
[Asserts]
jsonpath "$.error" contains "cannot be updated if device is in use"

# 8. Domain Validation: In-use device CANNOT be deleted (Expects 409 Conflict)
DELETE http://localhost:8080/devices/{{device_id}}
HTTP 409
[Asserts]
jsonpath "$.error" contains "in use devices cannot be deleted"

# 9. Transition state back to 'available'
PATCH http://localhost:8080/devices/{{device_id}}
Content-Type: application/json
{
  "state": "available"
}
HTTP 200
[Asserts]
jsonpath "$.state" == "available"

# 10. Delete the device now that it is available (Expects 204 No Content)
DELETE http://localhost:8080/devices/{{device_id}}
HTTP 204

# 11. Verify device was deleted (Expects 404 Not Found)
GET http://localhost:8080/devices/{{device_id}}
HTTP 404
