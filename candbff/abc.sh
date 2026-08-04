git pull --ff-only
bash image_build.sh
kubectl delete pods -l app=candbff -n cftp-test
