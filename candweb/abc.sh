git pull
bash image_build.sh
kubectl delete pods -l app=candweb -n cftp-test
sleep 5

