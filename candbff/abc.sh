git pull
bash image_build.sh
kubectl delete pods -l app=candbff -n cftp-test
sleep 5

