# Cleanup

This is a simple list of general "cleanup" items, a "tech debt" to-do list:

* Makefile should not cause `docker pull` until an actual target is invoked. Currently, even `make -n <target>` invokes `parse-pkgs.sh` which pulls down images.
* Making an existing target image should not rebuild it unless explicitly told to. Currently, `make live` will _always_ rebuild it, as opposed to seeing that it exists.
* [pillar](https://github.com/lf-edge/eve/tree/master/pkg/pillar) as documented [here](https://github.com/lf-edge/eve/blob/master/docs/COMMS.md) is a catch-all that contains many different utilities and services merged in a single image and therefore container. This needs to be separated properly for security and maintainability.

## ✅ Progress on Pillar Disaggregation

### Completed:
* **hardware-model service** - Extracted to `pkg/hardware-model/` as standalone microservice
  * Maintains full API compatibility with existing pillar service
  * Reduced attack surface and improved security boundaries
  * Independent build, test, and deployment capabilities
  
### Next Candidates for Extraction:
* `command` - Simple test/debug utility with minimal dependencies
* `client` - Device registration and controller communication (more complex)
* `diag` - Diagnostic utilities (medium complexity)

### Benefits Achieved:
1. **Security**: Reduced monolith size by extracting first service
2. **Maintainability**: Clear service boundaries and independent testing
3. **Deployability**: Services can be updated independently
4. **Development**: Faster iteration on individual services
