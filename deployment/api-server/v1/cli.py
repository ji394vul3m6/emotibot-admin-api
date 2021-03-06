#!/usr/bin/python

import errno
import shutil
import sys
import os
import subprocess
import argparse
import copy
import traceback

# worker-voice-emotion-analysis => w-v-emotion
VOICE_EMOTION_MINIMAL = ['mysql', 'rabbitmq', 'api-voice-emotion',
                         'w-v-emotion',
                         'mongo',
                         'worker-voice-emotion-statistic',
                         'voice_emotion_houta', 'nginx', 'authentication',
                         'consul']
VOICE_EMOTION_FULL = copy.deepcopy(VOICE_EMOTION_MINIMAL)
VOICE_EMOTION_FULL.extend(['netdata', 'phpmyadmin', 'kibana', 'elastic-closer', 'logstash', 'elasticsearch', 'kibana'])
BF_UBT_APIGW = ['nginx', 'bf-ubt-apigw']


def _create_folder(folder):
    # real_path = os.path.realpath(output_folder)
    if os.path.exists(folder):
        if not os.path.isdir(folder):
            print '%s is not a folder!' % (folder)
            sys.exit(1)
    else:
        try:
            os.makedirs(folder)
        except Exception as exp:
            print 'makedirs %s failed: %s' % (folder, exp)
            sys.exit(1)
    return os.path.realpath(folder)


def do_save(output_folder, compose_file):
    real_path = _create_folder(output_folder)
    # docker-compose -f docker-compose.yml config | grep image
    ps = subprocess.Popen(['docker-compose', '-f', compose_file, 'config'],
                          stdout=subprocess.PIPE)
    raw_ret = subprocess.check_output(['grep', 'image'], stdin=ps.stdout)
    ret_lines = raw_ret.split('\n')
    for line in ret_lines:
        if not line:
            continue
        line = line.split("image:")[1]
        line = line.strip()
        line = line.lstrip()
        # print line
        name = line.rsplit(":", 1)[0]
        dst = '%s.tar' % (os.path.join(real_path, os.path.basename(name)))
        cmd = "docker save %s -o %s" % (line, dst)
        print cmd
        subprocess.call(cmd.split())


def do_load(folder):
    if not os.path.isdir(folder):
        print '%s is not dir' % folder
        sys.exit(1)

    for f in os.listdir(folder):
        cmd = "docker load -i %s" % os.path.join(folder, f)
        print cmd
        subprocess.call(cmd.split(" "))


def do_destroy(compose_file):
    # Delete all containers
    # docker rm $(docker ps -a -q)
    # Delete all images
    # docker rmi $(docker images -q)
    cmd = 'docker-compose -f %s down --remove-orphans' % compose_file
    print cmd
    subprocess.call(cmd.split())
    ret = subprocess.check_output(['docker', 'images', '-q'])
    rets = ret.split('\n')
    for r in rets:
        if not r:
            continue
        cmd = 'docker rmi %s' % r
        subprocess.call(cmd.split())


def do_run(compose_file, env_file, services, depends, number, number_asr):
    '''
    1) copy env_file to .env
    2) compose comand:
        docker-compose -f ./docker-compose.yml pull ${service}
        docker-compose -f ./docker-compose.yml stop --remove-orphans ${service}
        docker-compose -f ./docker-compose.yml up --force-recreate
        --remove-orphans ${depends} -d ${scale} ${service}
    '''
    # mkdir for test env
    if env_file.endswith('test.env'):
        try:
            os.makedirs('/tmp/persistant_storage')
        except Exception as exp:
            if exp.errno != errno.EEXIST:
                print 'makedirs %s failed: %s' % ('/tmp/persistant_storage',
                                                  exp)
                sys.exit(1)

    # compose command: pull images
    # cmd = 'docker-compose -f %s pull --parallel %s' % (
    try:
        cmd = 'docker-compose -f %s pull %s' % (
            compose_file, ' '.join(n for n in services) if services else '')
        print '### exec cmd: [%s]' % cmd.strip()
        subprocess.check_call(cmd.strip().split(" "))
    except:
        print(traceback.format_exc())
        print("##############################################################")
        print("#  Docker pull fail, use local images to start the services  #")
        print("##############################################################\n\n\n")

    # compose command: remove previous service
    cmd = 'docker-compose -f %s rm -sf %s' % (
        compose_file, ' '.join(n for n in services) if services else '')
    print '### exec cmd: [%s]' % cmd.strip()
    subprocess.call(cmd.strip().split(" "))

    # TODO: deal with depends and scale
    no_deps = ''
    scale = ''
    scaleList = list()

    if services:
        if depends is False:
            no_deps = '--no-deps '
        for s in services:
            if s == 'w-v-emotion':
                scaleList.append('--scale %s=%s' % (s, number))
            elif s == 'w-v-asr':
                scaleList.append('--scale %s=%s' % (s, number_asr))

    scale_cmd = ' '.join(n for n in scaleList) if scaleList else ''
    scale_cmd = '' if not scale_cmd else '%s ' % scale_cmd

    cmd = 'docker-compose -f %s up --force-recreate --remove-orphans %s%s-d %s' % (
        compose_file,
        no_deps,
        scale_cmd,
        ' '.join(n for n in services) if services else '')
    print '### exec cmd: [%s]' % cmd.strip()
    subprocess.call(cmd.strip().split(" "))


def do_stop(compose_file, services):
    # docker-compose -f docker-compose.yml stop ${service}
    cmd = 'docker-compose -f %s stop %s' % (compose_file,
                                            ' '.join(n for n in services))
    print '### exec cmd: [%s]' % cmd.strip()
    subprocess.call(cmd.strip().split(" "))


def do_list(compose_file):
    cmd = 'docker-compose -f %s config --service' % compose_file
    print '### exec cmd: [%s]' % cmd.strip()
    subprocess.call(cmd.strip().split(" "))


def _do_copy_env(env_file):
    # copy env_file to .env
    dst_file = os.path.join(os.path.dirname(env_file), '.env')
    try:
        shutil.copyfile(env_file, dst_file)
    except Exception as exp:
        print 'copy %s to %s failed due to %s' % (env_file, dst_file, exp)
        sys.exit(1)


def main():
    # parse args
    parser = argparse.ArgumentParser()
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument(
        '--save',
        action='store_true',
        help='Save images. '
        'E.g. docker-compose --save -o ${/path/to/output_folder} -f ${YAML}')
    group.add_argument(
        '--load',
        action='store_true',
        help='Load images. '
        'E.g. docker-compose --load -o ${/path/to/output_folder}')
    group.add_argument(
        '--destroy',
        action='store_true',
        help='Destrop ALL images. '
        'E.g. docker-compose --destroy -f ${YAML}')
    group.add_argument(
        '--run',
        action='store_true',
        help='Run service. '
        'E.g. docker-compose --run -f ${YAML} -e ${ENV} -s ${service1} -s ${service2}')
    group.add_argument('--stop', action='store_true')
    group.add_argument('--list', action='store_true',
                       help='list all supported services')
    parser.add_argument('-g', '--service_group',
                        choices=['voice-emotion-full',
                                 'voice-emotion-min',
                                 'bf-ubt-apigw'],
                        default='voice-emotion-full')
    parser.add_argument('-o', '--image_folder', default='/tmp/api_srv_images')
    parser.add_argument('-f', '--compose_file', default='./docker-compose.yml')
    parser.add_argument('-e', '--env', required=True,
                        help='test.env or api-sh.env or api.env')
    parser.add_argument('-s', '--service', action='append', default=[])
    parser.add_argument('-d', '--depends', action='store_true', default=False,
                        help='if service is empty, depends always be true')
    parser.add_argument('-n', '--number', type=int, default=1,
                        help='only affect on voice analysis service')
    parser.add_argument('-n_asr', '--number_asr', type=int, default=2,
                        help='only affect on voice asr service')
    args = parser.parse_args()
    print args

    # copy env file as .env
    _do_copy_env(args.env)

    # add service of group if not give
    if not args.service:
        if getBoolFromEnvFile('WORKER_ENV_KEY_ASR_ENABLE', False) is True:
            args.service.extend(VOICE_EMOTION_FULL)
            args.service.append('w-v-asr')
            if getBoolFromEnvFile('WORKER_ENV_KEY_CALL_DIALOGUE_ANALYSIS_ENABLE', False) is True:
                args.service.append('call-analysis')

        elif args.service_group == 'voice-emotion-min':
            args.service.extend(VOICE_EMOTION_MINIMAL)
        elif args.service_group == 'bf-ubt-apigw':
            args.service.extend(BF_UBT_APIGW)
        elif args.service_group == 'voice-emotion-full':
            args.service.extend(VOICE_EMOTION_FULL)

    # do action
    if args.save:
        if not os.path.exists(args.compose_file):
            parser.print_help()
        do_save(args.image_folder, args.compose_file)
    elif args.load:
        do_load(args.image_folder)
    elif args.destroy:
        do_destroy(args.compose_file)
    elif args.run:
        if not os.path.exists(args.compose_file):
            parser.print_help()
        do_run(args.compose_file, args.env, args.service,
               args.depends, args.number, args.number_asr)
    elif args.stop:
        if not os.path.exists(args.compose_file):
            parser.print_help()
        do_stop(args.compose_file, args.service)
    elif args.list:
        if not os.path.exists(args.compose_file):
            parser.print_help()
        do_list(args.compose_file)
    else:
        parser.print_help()


def getBoolFromEnvFile(keyname, defaultValue):
    with open('.env', 'r') as fd:
        for line in fd:
            try:
                if line.find(keyname) != -1:
                    if line.split('=')[-1].strip(' \t\n\r').lower() in ['true', 't', '1']:
                        return True
                    else:
                        return False
            except:
                print(traceback.format_exc())
    return defaultValue


if __name__ == '__main__':
    main()
