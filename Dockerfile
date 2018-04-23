FROM alpine
EXPOSE 8000
ADD k8s-ingress-admission-controller /
CMD ["/k8s-ingress-admission-controller","--alsologtostderr","-v=4","2>&1"]
