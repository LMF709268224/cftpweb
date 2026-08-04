git pull --ff-only
bash image_build.sh
kubectl delete pods -l app=candweb -n cftp-test

