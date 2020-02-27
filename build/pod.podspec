Pod::Spec.new do |spec|
  spec.name         = 'Yocoin'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/Yocoin15/Yocoin_Sources'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'YoCoin command line client'
  spec.source       = { :git => 'https://github.com/Yocoin15/Yocoin_Sources.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Yocoin.framework'

	spec.prepare_command = <<-CMD
    curl https://gethstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Yocoin.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
