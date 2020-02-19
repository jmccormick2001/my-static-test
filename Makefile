NS = rq
IMAGEUSER = jemccorm
IMAGE = my-static-test
IMAGE_VERSION = v0.0.1
BUNDLE = /tmp/bundle.zip
compile:
	go build -o bin/$(IMAGE) ./pkg/static
build-image: compile
	sudo podman build --tag quay.io/$(IMAGEUSER)/$(IMAGE):$(IMAGE_VERSION) -f ./Dockerfile
push-image: 
	sudo podman push --authfile /home/jeffmc/.docker/config.json $(IMAGEUSER)/$(IMAGE):$(IMAGE_VERSION) docker://quay.io/$(IMAGEUSER)/$(IMAGE):$(IMAGE_VERSION)
clean:   
	rm ./bin/$(IMAGE)
rbac:   
	kubectl -n rq delete role,sa,rolebinding $(IMAGE)
	kubectl -n rq create -f manifests/role.yaml
	kubectl -n rq create -f manifests/role_binding.yaml
	kubectl -n rq create -f manifests/service_account.yaml
test:   
	rm $(BUNDLE)
	$ (cd  ./manifests/rqlite-operator; zip -r $(BUNDLE) . )
	kubectl -n rq delete configmap $(IMAGE) --ignore-not-found=true
	kubectl -n rq create configmap $(IMAGE) --from-file=bundle=$(BUNDLE) 
	kubectl -n rq delete pod $(IMAGE) --ignore-not-found=true
	kubectl -n rq create -f manifests/pod.yaml
run:   
	sudo podman run --name my-static-test --rm quay.io/jemccorm/my-static-test:v0.0.1

