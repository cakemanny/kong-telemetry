# vim: syntax=python filetype=tiltfile

docker_build('dice', 'dice')
docker_build('go-plugins', 'goplugin')

k8s_yaml(kustomize("."))

k8s_resource('kong', port_forwards=[8000, 8001])
