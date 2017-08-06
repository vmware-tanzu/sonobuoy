# Copyright 2017 Heptio Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

local k = import "ksonnet.beta.2/k.libsonnet";

local conf = {
    namespace: "heptio-sonobuoy",
    selector: {
        run: "sonobuoy-master",
    },
    labels: $.selector + {
        component: $.pod.name,
    },
    pod: {
        name: "sonobuoy",
        labels: $.labels + {
            tier: "analysis",
        },
        restartPolicy: "Never",
        serviceAccountName: "sonobuoy-serviceaccount",
    },
    service: {
        name: "sonobuoy-master",
        port: {
            port: 8080,
            protocol: "TCP",
            targetPort: 8080,
        },
        type: "ClusterIP",
    },
    master: {
        name: "kube-sonobuoy",
        command: ["/bin/bash", "-c", "/sonobuoy master --no-exit=true -v 3 --logtostderr"],
        image: "gcr.io/heptio-prod/sonobuoy:cha-0.0.1",
        imagePullPolicy: "Always",
        volumeMounts: [
            {
                name: $.volumes[0].name,
                mountPath: "/etc/sonobuoy",
            },
            {
                name: $.volumes[1].name,
                mountPath: "/etc/sonobuoy/plugins.d",
            },
            {
                name: $.volumes[2].name,
                mountPath: "/tmp/sonobuoy",
            },
        ],
    },
    volumes: [
        {
            name: "sonobuoy-config-volume",
            configMap: {name: "sonobuoy-config-cm"},
        },
        {
            name: "sonobuoy-plugins-volume",
            configMap: {name: "sonobuoy-plugins-cm"},
        },
        {
            name: "output-volume",
            hostPath: {path: "/tmp/sonobuoy"},
        },
    ],
    name: "sonobuoy",
};

local sonobuoyPod = local pod = k.core.v1.pod;
    pod.new() +
    pod.mixin.metadata.name(conf.pod.name) +
    pod.mixin.metadata.namespace(conf.namespace) +
    pod.mixin.metadata.labels(conf.pod.labels) +
    pod.mixin.spec.restartPolicy(conf.pod.restartPolicy) +
    pod.mixin.spec.serviceAccountName(conf.pod.serviceAccountName) +
    pod.mixin.spec.containers([
        conf.master +
            pod.mixin.spec.containersType.env([
               pod.mixin.spec.containersType.envType.fromFieldPath("SONOBUOY_ADVERTISE_IP", "status.podIP")
            ])
    ]) +
    pod.mixin.spec.volumes(conf.volumes);

local sonobuoyService = local svc = k.core.v1.service;
    svc.new(conf.service.name, conf.selector, [conf.service.port],) +
    svc.mixin.metadata.namespace(conf.namespace) +
    svc.mixin.metadata.labels(conf.labels) +
    svc.mixin.spec.type(conf.service.type);


local secret = {
    apiVersion: "v1",
    data: {
        ".dockercfg": "eyJnY3IuaW8vaGVwdGlvLXByb2QiOnsidXNlcm5hbWUiOiJfanNvbl9rZXkiLCJwYXNzd29yZCI6IntcbiAgXCJ0eXBlXCI6IFwic2VydmljZV9hY2NvdW50XCIsXG4gIFwicHJvamVjdF9pZFwiOiBcImhlcHRpby1wcm9kXCIsXG4gIFwicHJpdmF0ZV9rZXlfaWRcIjogXCI4NTUyODZhOGZmM2QwZTJmZGI5YTYzOWIxN2UwNGE1OWIxMmNjZTc5XCIsXG4gIFwicHJpdmF0ZV9rZXlcIjogXCItLS0tLUJFR0lOIFBSSVZBVEUgS0VZLS0tLS1cXG5NSUlFdlFJQkFEQU5CZ2txaGtpRzl3MEJBUUVGQUFTQ0JLY3dnZ1NqQWdFQUFvSUJBUUM3cHJlSXdOZy9jcThBXFxueWRnNStFK3dpcTVsWVhjZDRCVzVOVHpBTDRtS1JybUN4ZkkrSmlISVBBQzEyTjcxcXNkMHRvN05xdGFlQUhTQ1xcblRBT01xdStGcWc2cTN4MG5tYTdEUHNCK2hzWEE5N1lNOVJsTEV0b09zQXhVbUpIZHBMZTdJTVhrc0JKU0FOdzRcXG5aUHhuQzFQN3QzSHNhekozYnROcDVlQnJ3OFdDSHMzZXlveUNnWnRQZmZFWTh4aS9OdDlBdHZGUmRQSVRTbjUvXFxuT1lGVUdhY0xFVWdhd0xPUkoyYnVPWG82Ym1IS0VTWXcyeS9ra0tmRVFqenQ4N0xqazlHUzdlVHBFTWFFWWlrMFxcblpQTzNWUlp3RHFzUVN5bjcyV25RbHp5NUtCS2M1TFRsSDhTOEdxU0dZakl6ODFmbldmTktWajhaNVQ4eHF6R3BcXG5YMFFMRDE2akFnTUJBQUVDZ2dFQUlycnc5aEVIRlBzalU2aUg2ZmJBdlFKMTA1S3Q2cXdhS1N5bXdVVXJGaG9QXFxuNUpuQlEycG9Uemgzd3pCUDE3VVRkRkVEYmlKRDFYdHRGTjUvdTYyNVpnbzY2N29lbWNFeVhMV0ZDQVhtbk5mYlxcbjdGc0NvdWZxenRRYmZCNit4SUJqZHZGU1h3aU5ZM3NBUnptOWswNi83UE1mVlN2ajY4SHB4QXhGTkh6SDFkU1BcXG5qM09Eb0dnYmNlaVgzdzkyY0JOVjMwakRlVjI0OE5EVjIwWnJpWE5VN0tHM1JqejZpbC8xOWYvSFdGbkFTV3JjXFxud3hGRFozOUZwcWpUSVNLc1RtK0xHU2lWWVNzQy9LamI0L2JJWjR5Mmw3dE0zN01ieWE2dHkyeVcvcUZGOHhNT1xcbkxRbXh4Wk4wSmt6RGp4cFlsQU9ySWxhc2ZYS3RpRTlzQUl5QldoZUxQUUtCZ1FEZCtvZ2pNa2V2L3VETkYyNFNcXG4rTDhFT3lsQklHcEpRdnJQaUxRWWkwMUpyTjNwNXBoT3d6SU5SWnpaSkdqVjUyNUt2U3RQQ0FKYnBOcHI2cFN5XFxuSE1YTVFJUXI0NnpnNGNZbnpwY3FkZ2w5NnNqT1VHdy9NOHVrdUM4SUlSUWJqY0ZjdTcwdnRQSlZQT0N0LzY4YVxcbm9KR1NJK3hHUFJhTHZDOHU4OXlEcld2STd3S0JnUURZYVZRVjdVeXN0SFB5ZlREeTFRdmV1U05NMkV5VGdjQWhcXG5XbExTdjJOVEwzQTA3c243a0kveEFlendjTE1nVStnWEc3ZnVSTVY4Ty9qeFc3a2l0a2hEZG9nRTI4M1NsWFFOXFxuTXcxR1JGSkxOa21kNnhCTjdlV1RLRXFiU2dOdFcrdUdnZXo1c2lFQW53OGwrSG43R2xtdGVtc05WOGs1ZXNNRFxcbmdFaEgwYnQ5alFLQmdIYUcrZjVoTUtvbkhING1qcDRPdUdCWC9yVkp5N2NHenNuV0l1UWdMY3F6UldOSnYvWCtcXG5nRkZaNUdDRjhueVZNTzB6aVZhUDNrSjFDalFwYy9DUE1JYlp4RGx6UHJKdG05TlJtcUlQbVEzbE9nZ0FKV3l5XFxuQ1lFUTMvd2xQWWxnN2VqSVRrS1ZDZmk5b3ZNRjNjZ0lDUExSdjYzWm5KcE1oNTA0bFh5eU15VjlBb0dCQUo4M1xcbkhyM2pELzRmNVE0S1BQRHEvUWluZk9rVVdZSk1lMllPWmREWExlUU5pcWZtNW9OR0lDQllJbEhqR0dZNFZSQnJcXG5QQzc0T2JMbERJbkZ5YmRZRkdKTjJsUjR3anlqNS9XdUVaNFl0ZExQNWVsZy8yWFdHSWpqbzhBTCsrbUJMdzZPXFxubUNJOGd6dEp5b05OQTdGUitaZy84SEtvbTByR25xTDh4akRRaFBnaEFvR0FDVUUzOFFhYVk5M1l1Ri8vbHFzVFxcbnF6V1NRMXNuQzZGak1EL1NDaWRkV2tod2tQOGVENHBscDhyT3psVVJvV0VjV1hESlVuaXkyYjVZMEJnR3hEYXRcXG5ScGl3QmdDWEdWRU1RTlJldmJGUXBRVDZXcnpzRDJqNXJUbTJjdEhhN3RUK1E3VHNJWUkrZGpkZk9oZTdNNGpBXFxuc0lYTHZKNC9Pa1JKTURSU1pNdGlmTmM9XFxuLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLVxcblwiLFxuICBcImNsaWVudF9lbWFpbFwiOiBcIms4cy1nY3ItYXV0aC1yb0BoZXB0aW8tcHJvZC5pYW0uZ3NlcnZpY2VhY2NvdW50LmNvbVwiLFxuICBcImNsaWVudF9pZFwiOiBcIjEwMjMyMzU4Njc2MjEwMjQ3MDYyOFwiLFxuICBcImF1dGhfdXJpXCI6IFwiaHR0cHM6Ly9hY2NvdW50cy5nb29nbGUuY29tL28vb2F1dGgyL2F1dGhcIixcbiAgXCJ0b2tlbl91cmlcIjogXCJodHRwczovL2FjY291bnRzLmdvb2dsZS5jb20vby9vYXV0aDIvdG9rZW5cIixcbiAgXCJhdXRoX3Byb3ZpZGVyX3g1MDlfY2VydF91cmxcIjogXCJodHRwczovL3d3dy5nb29nbGVhcGlzLmNvbS9vYXV0aDIvdjEvY2VydHNcIixcbiAgXCJjbGllbnRfeDUwOV9jZXJ0X3VybFwiOiBcImh0dHBzOi8vd3d3Lmdvb2dsZWFwaXMuY29tL3JvYm90L3YxL21ldGFkYXRhL3g1MDkvazhzLWdjci1hdXRoLXJvJTQwaGVwdGlvLXByb2QuaWFtLmdzZXJ2aWNlYWNjb3VudC5jb21cIlxufSIsImVtYWlsIjoidXNlckBleGFtcGxlLmNvbSIsImF1dGgiOiJYMnB6YjI1ZmEyVjVPbnNLSUNBaWRIbHdaU0k2SUNKelpYSjJhV05sWDJGalkyOTFiblFpTEFvZ0lDSndjbTlxWldOMFgybGtJam9nSW1obGNIUnBieTF3Y205a0lpd0tJQ0FpY0hKcGRtRjBaVjlyWlhsZmFXUWlPaUFpT0RVMU1qZzJZVGhtWmpOa01HVXlabVJpT1dFMk16bGlNVGRsTURSaE5UbGlNVEpqWTJVM09TSXNDaUFnSW5CeWFYWmhkR1ZmYTJWNUlqb2dJaTB0TFMwdFFrVkhTVTRnVUZKSlZrRlVSU0JMUlZrdExTMHRMVnh1VFVsSlJYWlJTVUpCUkVGT1FtZHJjV2hyYVVjNWR6QkNRVkZGUmtGQlUwTkNTMk4zWjJkVGFrRm5SVUZCYjBsQ1FWRkROM0J5WlVsM1RtY3ZZM0U0UVZ4dWVXUm5OU3RGSzNkcGNUVnNXVmhqWkRSQ1Z6Vk9WSHBCVERSdFMxSnliVU40WmtrclNtbElTVkJCUXpFeVRqY3hjWE5rTUhSdk4wNXhkR0ZsUVVoVFExeHVWRUZQVFhGMUswWnhaelp4TTNnd2JtMWhOMFJRYzBJcmFITllRVGszV1UwNVVteE1SWFJ2VDNOQmVGVnRTa2hrY0V4bE4wbE5XR3R6UWtwVFFVNTNORnh1V2xCNGJrTXhVRGQwTTBoellYcEtNMkowVG5BMVpVSnlkemhYUTBoek0yVjViM2xEWjFwMFVHWm1SVms0ZUdrdlRuUTVRWFIyUmxKa1VFbFVVMjQxTDF4dVQxbEdWVWRoWTB4RlZXZGhkMHhQVWtveVluVlBXRzgyWW0xSVMwVlRXWGN5ZVM5cmEwdG1SVkZxZW5RNE4weHFhemxIVXpkbFZIQkZUV0ZGV1dsck1GeHVXbEJQTTFaU1duZEVjWE5SVTNsdU56SlhibEZzZW5rMVMwSkxZelZNVkd4SU9GTTRSM0ZUUjFscVNYbzRNV1p1VjJaT1MxWnFPRm8xVkRoNGNYcEhjRnh1V0RCUlRFUXhObXBCWjAxQ1FVRkZRMmRuUlVGSmNuSjNPV2hGU0VaUWMycFZObWxJTm1aaVFYWlJTakV3TlV0ME5uRjNZVXRUZVcxM1ZWVnlSbWh2VUZ4dU5VcHVRbEV5Y0c5VWVtZ3pkM3BDVURFM1ZWUmtSa1ZFWW1sS1JERllkSFJHVGpVdmRUWXlOVnBuYnpZMk4yOWxiV05GZVZoTVYwWkRRVmh0Yms1bVlseHVOMFp6UTI5MVpuRjZkRkZpWmtJMkszaEpRbXBrZGtaVFdIZHBUbGt6YzBGU2VtMDVhekEyTHpkUVRXWldVM1pxTmpoSWNIaEJlRVpPU0hwSU1XUlRVRnh1YWpOUFJHOUhaMkpqWldsWU0zYzVNbU5DVGxZek1HcEVaVll5TkRoT1JGWXlNRnB5YVZoT1ZUZExSek5TYW5vMmFXd3ZNVGxtTDBoWFJtNUJVMWR5WTF4dWQzaEdSRm96T1Vad2NXcFVTVk5MYzFSdEsweEhVMmxXV1ZOelF5OUxhbUkwTDJKSldqUjVNbXczZEUwek4wMWllV0UyZEhreWVWY3ZjVVpHT0hoTlQxeHVURkZ0ZUhoYVRqQkthM3BFYW5od1dXeEJUM0pKYkdGelpsaExkR2xGT1hOQlNYbENWMmhsVEZCUlMwSm5VVVJrSzI5bmFrMXJaWFl2ZFVST1JqSTBVMXh1SzB3NFJVOTViRUpKUjNCS1VYWnlVR2xNVVZscE1ERktjazR6Y0RWd2FFOTNla2xPVWxwNldrcEhhbFkxTWpWTGRsTjBVRU5CU21Kd1RuQnlObkJUZVZ4dVNFMVlUVkZKVVhJME5ucG5OR05aYm5wd1kzRmtaMnc1Tm5OcVQxVkhkeTlOT0hWcmRVTTRTVWxTVVdKcVkwWmpkVGN3ZG5SUVNsWlFUME4wTHpZNFlWeHViMHBIVTBrcmVFZFFVbUZNZGtNNGRUZzVlVVJ5VjNaSk4zZExRbWRSUkZsaFZsRldOMVY1YzNSSVVIbG1WRVI1TVZGMlpYVlRUazB5UlhsVVoyTkJhRnh1VjJ4TVUzWXlUbFJNTTBFd04zTnVOMnRKTDNoQlpYcDNZMHhOWjFVcloxaEhOMloxVWsxV09FOHZhbmhYTjJ0cGRHdG9SR1J2WjBVeU9ETlRiRmhSVGx4dVRYY3hSMUpHU2t4T2EyMWtObmhDVGpkbFYxUkxSWEZpVTJkT2RGY3JkVWRuWlhvMWMybEZRVzUzT0d3clNHNDNSMnh0ZEdWdGMwNVdPR3MxWlhOTlJGeHVaMFZvU0RCaWREbHFVVXRDWjBoaFJ5dG1OV2hOUzI5dVNFZzBiV3B3TkU5MVIwSllMM0pXU25rM1kwZDZjMjVYU1hWUloweGpjWHBTVjA1S2RpOVlLMXh1WjBaR1dqVkhRMFk0Ym5sV1RVOHdlbWxXWVZBemEwb3hRMnBSY0dNdlExQk5TV0phZUVSc2VsQnlTblJ0T1U1U2JYRkpVRzFSTTJ4UFoyZEJTbGQ1ZVZ4dVExbEZVVE12ZDJ4UVdXeG5OMlZxU1ZSclMxWkRabWs1YjNaTlJqTmpaMGxEVUV4U2RqWXpXbTVLY0Uxb05UQTBiRmg1ZVUxNVZqbEJiMGRDUVVvNE0xeHVTSEl6YWtRdk5HWTFVVFJMVUZCRWNTOVJhVzVtVDJ0VlYxbEtUV1V5V1U5YVpFUllUR1ZSVG1seFptMDFiMDVIU1VOQ1dVbHNTR3BIUjFrMFZsSkNjbHh1VUVNM05FOWlUR3hFU1c1R2VXSmtXVVpIU2s0eWJGSTBkMnA1YWpVdlYzVkZXalJaZEdSTVVEVmxiR2N2TWxoWFIwbHFhbTg0UVV3cksyMUNUSGMyVDF4dWJVTkpPR2Q2ZEVwNWIwNU9RVGRHVWl0YVp5ODRTRXR2YlRCeVIyNXhURGg0YWtSUmFGQm5hRUZ2UjBGRFZVVXpPRkZoWVZrNU0xbDFSaTh2YkhGelZGeHVjWHBYVTFFeGMyNUROa1pxVFVRdlUwTnBaR1JYYTJoM2ExQTRaVVEwY0d4d09ISlBlbXhWVW05WFJXTlhXRVJLVlc1cGVUSmlOVmt3UW1kSGVFUmhkRnh1VW5CcGQwSm5RMWhIVmtWTlVVNVNaWFppUmxGd1VWUTJWM0o2YzBReWFqVnlWRzB5WTNSSVlUZDBWQ3RSTjFSelNWbEpLMlJxWkdaUGFHVTNUVFJxUVZ4dWMwbFlUSFpLTkM5UGExSktUVVJTVTFwTmRHbG1UbU05WEc0dExTMHRMVVZPUkNCUVVrbFdRVlJGSUV0RldTMHRMUzB0WEc0aUxBb2dJQ0pqYkdsbGJuUmZaVzFoYVd3aU9pQWlhemh6TFdkamNpMWhkWFJvTFhKdlFHaGxjSFJwYnkxd2NtOWtMbWxoYlM1bmMyVnlkbWxqWldGalkyOTFiblF1WTI5dElpd0tJQ0FpWTJ4cFpXNTBYMmxrSWpvZ0lqRXdNak15TXpVNE5qYzJNakV3TWpRM01EWXlPQ0lzQ2lBZ0ltRjFkR2hmZFhKcElqb2dJbWgwZEhCek9pOHZZV05qYjNWdWRITXVaMjl2WjJ4bExtTnZiUzl2TDI5aGRYUm9NaTloZFhSb0lpd0tJQ0FpZEc5clpXNWZkWEpwSWpvZ0ltaDBkSEJ6T2k4dllXTmpiM1Z1ZEhNdVoyOXZaMnhsTG1OdmJTOXZMMjloZFhSb01pOTBiMnRsYmlJc0NpQWdJbUYxZEdoZmNISnZkbWxrWlhKZmVEVXdPVjlqWlhKMFgzVnliQ0k2SUNKb2RIUndjem92TDNkM2R5NW5iMjluYkdWaGNHbHpMbU52YlM5dllYVjBhREl2ZGpFdlkyVnlkSE1pTEFvZ0lDSmpiR2xsYm5SZmVEVXdPVjlqWlhKMFgzVnliQ0k2SUNKb2RIUndjem92TDNkM2R5NW5iMjluYkdWaGNHbHpMbU52YlM5eWIySnZkQzkyTVM5dFpYUmhaR0YwWVM5NE5UQTVMMnM0Y3kxblkzSXRZWFYwYUMxeWJ5VTBNR2hsY0hScGJ5MXdjbTlrTG1saGJTNW5jMlZ5ZG1salpXRmpZMjkxYm5RdVkyOXRJZ3A5In19"
    },
    kind: "Secret",
    metadata: {
        name: "heptio-prod-image-ro",
        namespace: "heptio-sonobuoy"
    },
    type: "kubernetes.io/dockercfg",
},

k.core.v1.list.new([secret, sonobuoyPod, sonobuoyService])
