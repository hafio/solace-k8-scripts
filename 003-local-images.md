# Using local tar images for kubectl setup

Refer to:
- https://medium.com/@tanmaybhandge/how-do-you-use-the-local-images-in-kubernetes-f5cbf375079c
- https://www.baeldung.com/ops/docker-local-images-minikube
- https://stackoverflow.com/questions/69981852/how-to-use-local-docker-images-in-kubernetes-deployments-not-minikube
- https://github.com/containerd/cri/blob/master/docs/crictl.md#directly-load-a-container-image
- https://blog.scottlowe.org/2020/01/25/manually-loading-container-images-with-containerd/

# Steps

1. Prepare container image (tar) file - refer to other guides to build/export/generate images
2. Transfer images to (all) worker nodes `scp <image file> <user>@<worker node>:<path>`
3. Login to worker node and run `sudo ctr -n k8s.io images import <image file> [--digests=true]` on every node
4. Verify images are imported `sudo ctr image ls` or `sudo crictl images`

# Minikube Steps

Instead of #2, run `minikube image load <image file> [-p <cluster name>]`. This will take some time but should add the image to all nodes.